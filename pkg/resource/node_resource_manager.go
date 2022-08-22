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
	coreinformers "k8s.io/client-go/informers/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"

	predictionv1 "github.com/gocrane/api/pkg/generated/informers/externalversions/prediction/v1alpha1"
	predictionlisters "github.com/gocrane/api/pkg/generated/listers/prediction/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/collector/types"
	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/metrics"
	"github.com/gocrane/crane/pkg/utils"
)

const (
	MinDeltaRatio                                 = 0.1
	StateExpiration                               = 1 * time.Minute
	TspUpdateInterval                             = 20 * time.Second
	TspNamespace                                  = "default"
	NodeReserveResourcePercentageAnnotationPrefix = "reserve.node.gocrane.io/%s"
)

var idToResourceMap = map[string]v1.ResourceName{
	v1.ResourceCPU.String():    v1.ResourceCPU,
	v1.ResourceMemory.String(): v1.ResourceMemory,
}

// ReserveResource is the cpu and memory reserve configuration
type ReserveResource struct {
	CpuPercent *float64
	MemPercent *float64
}

type NodeResourceManager struct {
	nodeName string
	client   clientset.Interface

	nodeLister corelisters.NodeLister
	nodeSynced cache.InformerSynced

	tspLister predictionlisters.TimeSeriesPredictionLister
	tspSynced cache.InformerSynced

	recorder record.EventRecorder

	stateChann chan map[string][]common.TimeSeries

	state map[string][]common.TimeSeries
	// Updated when get new data from stateChann, used to determine whether state has expired
	lastStateTime time.Time

	reserveResource ReserveResource

	tspName string
}

func NewNodeResourceManager(client clientset.Interface, nodeName string, nodeResourceReserved map[string]string, tspName string, nodeInformer coreinformers.NodeInformer,
	tspInformer predictionv1.TimeSeriesPredictionInformer, stateChann chan map[string][]common.TimeSeries) (*NodeResourceManager, error) {
	reserveCpuPercent, err := utils.ParsePercentage(nodeResourceReserved[v1.ResourceCPU.String()])
	if err != nil {
		return nil, err
	}
	reserveMemoryPercent, err := utils.ParsePercentage(nodeResourceReserved[v1.ResourceMemory.String()])
	if err != nil {
		return nil, err
	}

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: client.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "crane-agent"})

	o := &NodeResourceManager{
		nodeName:   nodeName,
		client:     client,
		nodeLister: nodeInformer.Lister(),
		nodeSynced: nodeInformer.Informer().HasSynced,
		tspLister:  tspInformer.Lister(),
		tspSynced:  tspInformer.Informer().HasSynced,
		recorder:   recorder,
		stateChann: stateChann,
		reserveResource: ReserveResource{
			CpuPercent: &reserveCpuPercent,
			MemPercent: &reserveMemoryPercent,
		},
		tspName: tspName,
	}
	return o, nil
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
	node := o.getNode()
	if len(node.Status.Addresses) == 0 {
		klog.Error("Node addresses is empty")
		return
	}
	nodeCopy := node.DeepCopy()

	resourcesFrom := o.BuildNodeStatus(nodeCopy)
	if !equality.Semantic.DeepEqual(&node.Status, &nodeCopy.Status) {
		// Update Node status extend-resource info
		// TODO fix: strategic merge patch kubernetes
		if _, err := o.client.CoreV1().Nodes().UpdateStatus(context.TODO(), nodeCopy, metav1.UpdateOptions{}); err != nil {
			klog.Errorf("Failed to update node %s extended resource, %v", nodeCopy.Name, err)
			return
		}
		klog.V(2).Infof("Update node %s extended resource successfully", node.Name)
		o.recorder.Event(node, v1.EventTypeNormal, "UpdateNode", generateUpdateEventMessage(resourcesFrom))
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
		return false, fmt.Errorf("tsp %s target is not specified", tsp.Name)
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

func (o *NodeResourceManager) BuildNodeStatus(node *v1.Node) map[string]int64 {
	tspCanNotBeReclaimedResource := o.GetCanNotBeReclaimedResourceFromTsp(node)
	localCanNotBeReclaimedResource := o.GetCanNotBeReclaimedResourceFromLocal()
	reserveCpuPercent := o.reserveResource.CpuPercent
	if nodeReserveCpuPercent, ok := getReserveResourcePercentFromNodeAnnotations(node.GetAnnotations(), v1.ResourceCPU.String()); ok {
		reserveCpuPercent = &nodeReserveCpuPercent
	}

	extResourceFrom := map[string]int64{}

	for resourceName, value := range tspCanNotBeReclaimedResource {
		resourceFrom := "tsp"
		maxUsage := value
		if localCanNotBeReclaimedResource[resourceName] > maxUsage {
			maxUsage = localCanNotBeReclaimedResource[resourceName]
			resourceFrom = "local"
		}

		var nextRecommendation float64
		switch resourceName {
		case v1.ResourceCPU:
			if reserveCpuPercent != nil {
				nextRecommendation = float64(node.Status.Allocatable.Cpu().Value()) - float64(node.Status.Allocatable.Cpu().Value())*(*reserveCpuPercent) - maxUsage/1000
			} else {
				nextRecommendation = float64(node.Status.Allocatable.Cpu().Value()) - maxUsage/1000
			}
		case v1.ResourceMemory:
			// unit of memory in prometheus is in Ki, need to be converted to byte
			nextRecommendation = float64(node.Status.Allocatable.Memory().Value()) - (maxUsage * 1000)
		default:
			continue
		}
		if nextRecommendation < 0 {
			nextRecommendation = 0
		}
		metrics.UpdateNodeResourceRecommendedValue(metrics.SubComponentNodeResource, metrics.StepGetExtResourceRecommended, string(resourceName), resourceFrom, nextRecommendation)
		extResourceName := fmt.Sprintf(utils.ExtResourcePrefixFormat, string(resourceName))
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

		extResourceFrom[resourceFrom+"-"+resourceName.String()] = int64(nextRecommendation)
	}

	return extResourceFrom
}

func (o *NodeResourceManager) GetCanNotBeReclaimedResourceFromTsp(node *v1.Node) map[v1.ResourceName]float64 {
	canNotBeReclaimedResource := map[v1.ResourceName]float64{
		v1.ResourceCPU:    0,
		v1.ResourceMemory: 0,
	}

	tsp, err := o.tspLister.TimeSeriesPredictions(TspNamespace).Get(o.tspName)
	if err != nil {
		klog.Errorf("Failed to get tsp: %#v", err)
		return canNotBeReclaimedResource
	}

	tspMatched, err := o.FindTargetNode(tsp, node.Status.Addresses)
	if err != nil {
		klog.Error(err.Error())
		return canNotBeReclaimedResource
	}

	if !tspMatched {
		klog.Errorf("Found tsp %s, but tsp not matched to node %s", o.tspName, node.Name)
		return canNotBeReclaimedResource
	}

	// build node status
	nextPredictionResourceStatus := &tsp.Status
	for _, predictionMetric := range nextPredictionResourceStatus.PredictionMetrics {
		resourceName, exists := idToResourceMap[predictionMetric.ResourceIdentifier]
		if !exists {
			continue
		}
		for _, timeSeries := range predictionMetric.Prediction {
			var nextUsage float64
			var nextUsageFloat float64
			var err error
			for _, sample := range timeSeries.Samples {
				if nextUsageFloat, err = strconv.ParseFloat(sample.Value, 64); err != nil {
					klog.Errorf("Failed to parse extend resource value %v: %v", sample.Value, err)
					continue
				}
				nextUsage = nextUsageFloat
				if canNotBeReclaimedResource[resourceName] < nextUsage {
					canNotBeReclaimedResource[resourceName] = nextUsage
				}
			}
		}
	}
	return canNotBeReclaimedResource
}

func (o *NodeResourceManager) GetCanNotBeReclaimedResourceFromLocal() map[v1.ResourceName]float64 {
	return map[v1.ResourceName]float64{
		v1.ResourceCPU:    o.GetCpuCoreCanNotBeReclaimedFromLocal(),
		v1.ResourceMemory: 0,
	}
}

func (o *NodeResourceManager) GetCpuCoreCanNotBeReclaimedFromLocal() float64 {
	if o.lastStateTime.Before(time.Now().Add(-20 * time.Second)) {
		klog.V(1).Infof("NodeResourceManager local state has expired")
		return 0
	}

	nodeCpuUsageTotalTimeSeries, ok := o.state[string(types.MetricNameCpuTotalUsage)]
	if !ok {
		klog.V(4).Infof("Can't get %s from NodeResourceManager local state, please make sure cpu metrics collector is defined in NodeQOS.", types.MetricNameCpuTotalUsage)
		return 0
	}
	nodeCpuUsageTotal := nodeCpuUsageTotalTimeSeries[0].Samples[0].Value

	var extResContainerCpuUsageTotal float64 = 0
	extResContainerCpuUsageTotalTimeSeries, ok := o.state[string(types.MetricNameExtResContainerCpuTotalUsage)]
	if ok {
		extResContainerCpuUsageTotal = extResContainerCpuUsageTotalTimeSeries[0].Samples[0].Value * 1000
	} else {
		klog.V(4).Infof("Can't get %s from NodeResourceManager local state", types.MetricNameExtResContainerCpuTotalUsage)
	}

	var exclusiveCPUIdle float64 = 0
	exclusiveCPUIdleTimeSeries, ok := o.state[string(types.MetricNameExclusiveCPUIdle)]
	if ok {
		exclusiveCPUIdle = exclusiveCPUIdleTimeSeries[0].Samples[0].Value
	} else {
		klog.V(4).Infof("Can't get %s from NodeResourceManager local state", types.MetricNameExclusiveCPUIdle)
	}

	klog.V(6).Infof("nodeCpuUsageTotal: %f, exclusiveCPUIdle: %f, extResContainerCpuUsageTotal: %f", nodeCpuUsageTotal, exclusiveCPUIdle, extResContainerCpuUsageTotal)

	// 1. Exclusive tethered CPU cannot be reclaimed even if the free part is free, so add the exclusive CPUIdle to the CanNotBeReclaimed CPU
	// 2. The CPU used by extRes-container needs to be reclaimed, otherwise it will be double-counted due to the allotted mechanism of k8s, so the extResContainerCpuUsageTotal is subtracted from the CanNotBeReclaimedCpu
	nodeCpuCannotBeReclaimedSeconds := nodeCpuUsageTotal + exclusiveCPUIdle - extResContainerCpuUsageTotal
	metrics.UpdateNodeCpuCannotBeReclaimedSeconds(nodeCpuCannotBeReclaimedSeconds)
	return nodeCpuCannotBeReclaimedSeconds
}

func getReserveResourcePercentFromNodeAnnotations(annotations map[string]string, resourceName string) (float64, bool) {
	if annotations == nil {
		return 0, false
	}
	var reserveResourcePercentStr string
	var ok = false
	switch resourceName {
	case v1.ResourceCPU.String():
		reserveResourcePercentStr, ok = annotations[fmt.Sprintf(NodeReserveResourcePercentageAnnotationPrefix, v1.ResourceCPU.String())]
	case v1.ResourceMemory.String():
		reserveResourcePercentStr, ok = annotations[fmt.Sprintf(NodeReserveResourcePercentageAnnotationPrefix, v1.ResourceMemory.String())]
	default:
	}
	if !ok {
		return 0, false
	}

	reserveResourcePercent, err := utils.ParsePercentage(reserveResourcePercentStr)
	if err != nil {
		return 0, false
	}

	return reserveResourcePercent, ok
}

func generateUpdateEventMessage(resourcesFrom map[string]int64) string {
	message := ""
	for k, v := range resourcesFrom {
		message = message + fmt.Sprintf("Updating elastic resource %s with %d.", k, v)
	}
	return message
}
