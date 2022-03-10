package offline

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	info "github.com/google/cadvisor/info/v1"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"k8s.io/klog/v2"

	predictionv1 "github.com/gocrane/api/pkg/generated/informers/externalversions/prediction/v1alpha1"
	predictionlisters "github.com/gocrane/api/pkg/generated/listers/prediction/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/collector/cadvisor"
	"github.com/gocrane/crane/pkg/ensurance/collector/types"
	stypes "github.com/gocrane/crane/pkg/ensurance/collector/types"
	"github.com/gocrane/crane/pkg/ensurance/executor"
	cgrpc "github.com/gocrane/crane/pkg/ensurance/grpc"
	cruntime "github.com/gocrane/crane/pkg/ensurance/runtime"
	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/utils"
)

const (
	ExtResourcePrefixFormat = "ext-resource.node.gocrane.io/%s"
	MinDeltaRatio           = 0.1
	StateExpiration         = 1 * time.Minute
)

type OfflineManager struct {
	nodeName string
	client   clientset.Interface

	podLister corelisters.PodLister
	podSynced cache.InformerSynced

	nodeLister corelisters.NodeLister
	nodeSynced cache.InformerSynced

	tspLister predictionlisters.TimeSeriesPredictionLister
	tspSynced cache.InformerSynced

	runtimeClient pb.RuntimeServiceClient
	runtimeConn   *grpc.ClientConn
	stateChann    chan map[string][]common.TimeSeries
	state         map[string][]common.TimeSeries
	lastStateTime time.Time

	collectors *sync.Map
}

func NewOfflineManager(client clientset.Interface, nodeName string, podInformer coreinformers.PodInformer, nodeInformer coreinformers.NodeInformer,
	tspInformer predictionv1.TimeSeriesPredictionInformer, runtimeEndpoint string, stateChann chan map[string][]common.TimeSeries, Collectors *sync.Map) *OfflineManager {
	runtimeClient, runtimeConn, err := cruntime.GetRuntimeClient(runtimeEndpoint, true)
	if err != nil {
		klog.Errorf("GetRuntimeClient failed %s", err.Error())
		return nil
	}

	o := &OfflineManager{
		nodeName:      nodeName,
		client:        client,
		podLister:     podInformer.Lister(),
		podSynced:     podInformer.Informer().HasSynced,
		nodeLister:    nodeInformer.Lister(),
		nodeSynced:    nodeInformer.Informer().HasSynced,
		tspLister:     tspInformer.Lister(),
		tspSynced:     tspInformer.Informer().HasSynced,
		runtimeClient: runtimeClient,
		runtimeConn:   runtimeConn,
		stateChann:    stateChann,
		collectors:    Collectors,
	}
	tspInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: o.reconcileTsp,
		UpdateFunc: func(old, cur interface{}) {
			o.reconcileTsp(cur)
		},
		DeleteFunc: o.reconcileTsp,
	})
	podInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			switch t := obj.(type) {
			case *v1.Pod:
				return ownedPod(t, o.nodeName)
			default:
				utilruntime.HandleError(fmt.Errorf("unable to handle object %T", obj))
				return false
			}
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: o.reconcilePod,
			UpdateFunc: func(old, cur interface{}) {
				o.reconcilePod(cur)
			},
			DeleteFunc: o.reconcilePod,
		},
	})
	return o
}

func (o *OfflineManager) Name() string {
	return "OfflineManager"
}

func (o *OfflineManager) Run(stop <-chan struct{}) {
	klog.Infof("Starting offline manager.")

	// Wait for the caches to be synced before starting workers
	if !cache.WaitForNamedCacheSync("offline-manager",
		stop,
		o.podSynced,
		o.tspSynced,
		o.nodeSynced,
	) {
		return
	}

	go func() {
		for {
			select {
			case state := <-o.stateChann:
				{
					o.state = state
					o.lastStateTime = time.Now()
				}
			case <-stop:
				klog.Infof("offline manager exit")
				if err := cgrpc.CloseGrpcConnection(o.runtimeConn); err != nil {
					klog.Errorf("Failed to close grpc connection: %v", err)
				}
				return
			}
		}
	}()

	return
}

func (o *OfflineManager) reconcileTsp(obj interface{}) {
	tsp, ok := obj.(*predictionapi.TimeSeriesPrediction)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		tsp, ok = tombstone.Obj.(*predictionapi.TimeSeriesPrediction)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Tombstone contained object that is not a tsp %#v", obj))
			return
		}
	}

	// get current node info
	target := tsp.Spec.TargetRef
	if target.Kind != config.TargetKindNode {
		return
	}
	node, err, retry := o.FindTargetNode(tsp)
	if err != nil {
		if !retry {
			return
		} else {
			klog.Errorf("FindTargetNode err : %#v", err)
			return
		}
	}

	nodeCopy := node.DeepCopy()
	o.BuildNodeStatus(tsp, nodeCopy)
	if !equality.Semantic.DeepEqual(&node.Status, &nodeCopy.Status) {
		// update Node status extend-resource info
		// TODO fix: strategic merge patch kubernetes
		if _, err := o.client.CoreV1().Nodes().Update(context.TODO(), nodeCopy, metav1.UpdateOptions{}); err != nil {
			klog.Errorf("Failed to update node %s's status extend-resource, %v", nodeCopy.Name, err)
			return
		}
		klog.V(4).Infof("Update Node %s Extend Resource Success according to TSP %s", node.Name, tsp.Name)
	}
	return
}

func (o *OfflineManager) FindTargetNode(tsp *predictionapi.TimeSeriesPrediction) (*v1.Node, error, bool) {
	address := tsp.Spec.TargetRef.Name
	if address == "" {
		return nil, fmt.Errorf("target is not specified"), false
	}

	node, err := o.nodeLister.Get(o.nodeName)
	if err != nil {
		klog.Errorf("Failed to get node: %v", err)
		return nil, err, true
	}

	// the reason we use node ip instead of node name as the target name is
	// some monitoring system does not persist node name
	for _, addr := range node.Status.Addresses {
		if addr.Address == address {
			return node, nil, false
		}
	}
	return nil, fmt.Errorf("target [%s] not found", address), false
}

func (o *OfflineManager) BuildNodeStatus(tsp *predictionapi.TimeSeriesPrediction, node *v1.Node) {
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
			if nextRecommendation <= 0 {
				klog.V(4).Infof("Unexpected recommendation,nodeName %s, maxUsage %v, nextRecommendation %v", node.Name, maxUsage, nextRecommendation)
				continue
			}
			extResourceName := fmt.Sprintf(ExtResourcePrefixFormat, string(*resourceName))
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

func (o *OfflineManager) reconcilePod(obj interface{}) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		pod, ok = tombstone.Obj.(*corev1.Pod)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Tombstone contained object that is not a pod %#v", obj))
			return
		}
	}

	o.updatePodExtResToCgroup(pod)
}

func ownedPod(pod *v1.Pod, nodeName string) bool {
	return pod.Spec.NodeName == nodeName && pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed
}

func (o *OfflineManager) updatePodExtResToCgroup(pod *corev1.Pod) {
	for _, c := range pod.Spec.Containers {
		for res, val := range c.Resources.Limits {
			if strings.HasPrefix(res.String(), fmt.Sprintf(ExtResourcePrefixFormat, v1.ResourceCPU)) {
				containerId := getContainerIdFromPod(pod, c.Name)
				if containerId == "" {
					continue
				}
				containerPeriod := o.getCPUPeriod(pod, containerId)
				if containerPeriod == 0 {
					continue
				}
				err := cruntime.UpdateContainerResources(o.runtimeClient, containerId, cruntime.UpdateOptions{CPUQuota: int64(float64(val.MilliValue()) / executor.CpuQuotaCoefficient * containerPeriod)})
				if err != nil {
					klog.Errorf("Failed to update pod %s container %s Resource, err %s", pod.Name, containerId, err.Error())
					continue
				}
			}
		}
	}
}

func getContainerIdFromPod(pod *corev1.Pod, containerName string) string {
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Name == containerName {
			return utils.GetContainerIdFromKey(cs.ContainerID)
		}
	}
	return ""
}

func (o *OfflineManager) getCPUPeriod(pod *corev1.Pod, containerId string) float64 {
	value, exists := o.collectors.Load(types.CadvisorCollectorType)
	now := time.Now()

	if exists && o.state != nil && !now.After(o.lastStateTime.Add(StateExpiration)) {
		_, containerCPUPeriods := executor.GetPodUsage(string(stypes.MetricNameContainerCpuPeriod), o.state, pod)
		for _, period := range containerCPUPeriods {
			if period.ContainerId == containerId {
				return period.Value
			}
		}
	}

	c := value.(cadvisor.CadvisorCollector)
	var query = info.ContainerInfoRequest{}
	containerInfoV1, err := c.Manager.GetContainerInfo(containerId, &query)
	if err != nil {
		klog.Errorf("ContainerInfoRequest failed: %v", err)
		return 0.0
	}
	return float64(containerInfoV1.Spec.Cpu.Period)
}
