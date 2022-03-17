package resource

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	coreinformers "k8s.io/client-go/informers/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	predictionv1 "github.com/gocrane/api/pkg/generated/informers/externalversions/prediction/v1alpha1"
	predictionlisters "github.com/gocrane/api/pkg/generated/listers/prediction/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/metrics"
	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/utils"
)

const (
	MinDeltaRatio     = 0.1
	StateExpiration   = 1 * time.Minute
	TspUpdateInterval = 20 * time.Second
)

type NodeResourceManager struct {
	nodeName string
	client   clientset.Interface

	nodeLister corelisters.NodeLister
	nodeSynced cache.InformerSynced

	tspLister predictionlisters.TimeSeriesPredictionLister
	tspSynced cache.InformerSynced

	stateChann chan map[string][]common.TimeSeries

	//TODO: use state to be a backup of tsp
	// A copy of data from stateChann
	state map[string][]common.TimeSeries
	// Updated when get new data from stateChann, used to determine whether state has expired
	lastStateTime time.Time
}

func NewNodeResourceManager(client clientset.Interface, nodeName string, podInformer coreinformers.PodInformer, nodeInformer coreinformers.NodeInformer,
	tspInformer predictionv1.TimeSeriesPredictionInformer, runtimeEndpoint string, stateChann chan map[string][]common.TimeSeries) *NodeResourceManager {
	o := &NodeResourceManager{
		nodeName:   nodeName,
		client:     client,
		nodeLister: nodeInformer.Lister(),
		nodeSynced: nodeInformer.Informer().HasSynced,
		tspLister:  tspInformer.Lister(),
		tspSynced:  tspInformer.Informer().HasSynced,
		stateChann: stateChann,
	}
	return o
}

func (o *NodeResourceManager) Name() string {
	return "NodeResourceManager"
}

func (o *NodeResourceManager) Run(stop <-chan struct{}) {
	klog.Infof("Starting node resource manager.")

	// Wait for the caches to be synced before starting workers
	if !cache.WaitForNamedCacheSync("node-resource-manager",
		stop,
		o.tspSynced,
		o.nodeSynced,
	) {
		return
	}

	go func() {
		tspUpdateTicker := time.NewTicker(TspUpdateInterval)
		defer tspUpdateTicker.Stop()
		for {
			select {
			case state := <-o.stateChann:
				o.state = state
				o.lastStateTime = time.Now()
			case <-tspUpdateTicker.C:
				start := time.Now()
				metrics.UpdateLastTime(string(known.ModuleNodeResourceManager), metrics.StepUpdateNodeResource, start)
				o.UpdateNodeResource()
				metrics.UpdateDurationFromStart(string(known.ModuleNodeResourceManager), metrics.StepUpdateNodeResource, start)
			case <-stop:
				klog.Infof("node resource manager exit")
				return
			}
		}
	}()

	return
}

func (o *NodeResourceManager) UpdateNodeResource() {
	tsps, err := o.tspLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list tsp: %#v", err)
		return
	}

	node := o.getNode()
	if len(node.Status.Addresses) == 0 {
		klog.Error("Node addresses is empty")
		return
	}
	nodeCopy := node.DeepCopy()

	for _, tsp := range tsps {
		// Get current node info
		target := tsp.Spec.TargetRef
		if target.Kind != config.TargetKindNode {
			return
		}

		// Whether tsp is matched with this node
		tspMatched, err := o.FindTargetNode(tsp, node.Status.Addresses)
		if err != nil {
			klog.Error(err.Error())
		}

		if !tspMatched {
			return
		}

		o.BuildNodeStatus(tsp, nodeCopy)
		if !equality.Semantic.DeepEqual(&node.Status, &nodeCopy.Status) {
			// Update Node status extend-resource info
			// TODO fix: strategic merge patch kubernetes
			if _, err := o.client.CoreV1().Nodes().Update(context.TODO(), nodeCopy, metav1.UpdateOptions{}); err != nil {
				klog.Errorf("Failed to update node %s's status extend-resource, %v", nodeCopy.Name, err)
				return
			}
			klog.V(4).Infof("Update Node %s Extend Resource Success according to TSP %s", node.Name, tsp.Name)
		}
		return
	}
}

func (o *NodeResourceManager) getNode() *v1.Node {
	node, err := o.nodeLister.Get(o.nodeName)
	if err != nil {
		klog.Errorf("Failed to get node: %v", err)
		return nil
	}
	return node
}

func (o *NodeResourceManager) FindTargetNode(tsp *predictionapi.TimeSeriesPrediction, addresses []v1.NodeAddress) (bool, error) {
	address := tsp.Spec.TargetRef.Name
	if address == "" {
		return false, fmt.Errorf("Tsp %s target is not specified", tsp.Name)
	}

	// the reason we use node ip instead of node name as the target name is
	// some monitoring system does not persist node name
	for _, addr := range addresses {
		if addr.Address == address {
			return true, nil
		}
	}
	klog.V(4).Infof("Target %s mismatch this node", address)
	return false, nil
}

func (o *NodeResourceManager) BuildNodeStatus(tsp *predictionapi.TimeSeriesPrediction, node *v1.Node) {
	idToResourceMap := map[string]*v1.ResourceName{}
	for _, metrics := range tsp.Spec.PredictionMetrics {
		if metrics.ResourceQuery == nil {
			continue
		}
		idToResourceMap[metrics.ResourceIdentifier] = metrics.ResourceQuery
	}
	// build node status
	nextPredictionResourceStatus := &tsp.Status
	for _, metrics := range nextPredictionResourceStatus.PredictionMetrics {
		resourceName, exists := idToResourceMap[metrics.ResourceIdentifier]
		if !exists {
			continue
		}
		for _, timeSeries := range metrics.Prediction {
			var maxUsage, nextUsage float64
			var nextUsageFloat float64
			var err error
			for _, sample := range timeSeries.Samples {
				if nextUsageFloat, err = strconv.ParseFloat(sample.Value, 64); err != nil {
					klog.Errorf("Failed to parse extend resource value %v: %v", sample.Value, err)
					continue
				}
				nextUsage = nextUsageFloat
				if maxUsage < nextUsage {
					maxUsage = nextUsage
				}
			}
			var nextRecommendation float64
			switch *resourceName {
			case v1.ResourceCPU:
				// cpu need to be scaled to m as ext resource cannot be decimal
				nextRecommendation = (float64(node.Status.Allocatable.Cpu().Value()) - maxUsage) * 1000
			case v1.ResourceMemory:
				// unit of memory in prometheus is in Ki, need to be converted to byte
				nextRecommendation = float64(node.Status.Allocatable.Memory().Value()) - (maxUsage * 1000)
			default:
				continue
			}
			if nextRecommendation < 0 {
				klog.V(4).Infof("Unexpected recommendation,nodeName %s, maxUsage %v, nextRecommendation %v", node.Name, maxUsage, nextRecommendation)
				continue
			}
			extResourceName := fmt.Sprintf(utils.ExtResourcePrefixFormat, string(*resourceName))
			resValue, exists := node.Status.Capacity[v1.ResourceName(extResourceName)]
			if exists && resValue.Value() != 0 &&
				math.Abs(float64(resValue.Value())-
					nextRecommendation)/float64(resValue.Value()) <= MinDeltaRatio {
				continue
			}
			node.Status.Capacity[v1.ResourceName(extResourceName)] =
				*resource.NewQuantity(int64(nextRecommendation), resource.DecimalSI)
			node.Status.Allocatable[v1.ResourceName(extResourceName)] =
				*resource.NewQuantity(int64(nextRecommendation), resource.DecimalSI)
		}
	}
}
