package resource

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
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
	NodeReserveResourcePercentageAnnotationPrefix = "reserve.node.gocrane.io/%s"
)

var idToResourceMap = map[string]v1.ResourceName{
	v1.ResourceCPU.String():    v1.ResourceCPU,
	v1.ResourceMemory.String(): v1.ResourceMemory,
}

// ReserveResource is the cpu and memory reserve configuration
type ReservedResource struct {
	CpuPercent float64
	MemPercent float64
}

type NodeResourceManager struct {
	nodeName         string
	tspName          string
	reservedResource ReservedResource

	client     clientset.Interface
	recorder   record.EventRecorder
	nodeLister corelisters.NodeLister
	nodeSynced cache.InformerSynced
	tspLister  predictionlisters.TimeSeriesPredictionLister
	tspSynced  cache.InformerSynced

	stateChann     chan map[string][]common.TimeSeries
	resourceStatus *known.ResourceStatus
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
		reservedResource: ReservedResource{
			CpuPercent: reserveCpuPercent,
			MemPercent: reserveMemoryPercent,
		},
		tspName: tspName,
	}
	return o, nil
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
		for {
			select {
			case state := <-o.stateChann:
				start := time.Now()
				metrics.UpdateLastTime(string(known.ModuleNodeResourceManager), metrics.StepUpdateNodeResource, start)
				if err := o.computeResourceStatus(state); err != nil {
					klog.ErrorS(err, "build resource status failed")
					continue
				}
				if err := o.updateResource(); err != nil {
					klog.ErrorS(err, "build resource status failed")
					continue
				}
				metrics.UpdateDurationFromStart(string(known.ModuleNodeResourceManager), metrics.StepUpdateNodeResource, start)
			case <-stop:
				klog.Infof("node resource manager exit")
				return
			}
		}
	}()

	return
}

func (o *NodeResourceManager) computeResourceStatus(tsm map[string][]common.TimeSeries) error {
	rs := &known.ResourceStatus{}
	transform := func(name types.MetricName) (int64, error) {
		if series, ok := tsm[string(name)]; !ok {
			return 0, fmt.Errorf("series %s missed", name)
		} else {
			return int64(series[0].Samples[0].Value), nil
		}
	}
	// 1. Get resource usage
	if val, err := transform(types.MetricNameCpuTotalUsage); err != nil {
		return err
	} else {
		rs.CPUUsage = resource.NewMilliQuantity(val, resource.DecimalSI)
	}
	if val, err := transform(types.MetricNameExtResContainerCpuTotalUsage); err != nil {
		return err
	} else {
		rs.CPUUsageOffline = resource.NewQuantity(val, resource.DecimalSI)
	}
	if val, err := transform(types.MetricNameExclusiveCPUIdle); err != nil {
		return err
	} else {
		rs.CPUSetIdle = resource.NewMilliQuantity(val, resource.DecimalSI)
	}
	if val, err := transform(types.MetricNameMemoryTotalUsage); err != nil {
		return err
	} else {
		rs.MemoryUsage = resource.NewQuantity(val, resource.BinarySI)
	}
	if val, err := transform(types.MetricNameExtResContainerMemTotalUsage); err != nil {
		return err
	} else {
		rs.MemoryUsageOffline = resource.NewQuantity(val, resource.BinarySI)
	}

	// 2. Get resource reserved
	node, err := o.getNode()
	if err != nil {
		return err
	}
	reservedCPUPercent := o.reservedResource.CpuPercent
	if nodeReserveCpuPercent, ok := getReserveResourcePercentFromNodeAnnotations(node.GetAnnotations(), v1.ResourceCPU.String()); ok {
		reservedCPUPercent = nodeReserveCpuPercent
	}
	reservedMemoryPercent := o.reservedResource.MemPercent
	if nodeReserveMemPercent, ok := getReserveResourcePercentFromNodeAnnotations(node.GetAnnotations(), v1.ResourceMemory.String()); ok {
		reservedMemoryPercent = nodeReserveMemPercent
	}
	rs.CPUReserved = resource.NewQuantity(int64(node.Status.Allocatable.Cpu().AsApproximateFloat64()*reservedCPUPercent), resource.DecimalSI)
	rs.MemoryReserved = resource.NewQuantity(int64(node.Status.Allocatable.Memory().AsApproximateFloat64()*reservedMemoryPercent), resource.DecimalSI)

	// 3. Get resource from TSP
	onlineResourceFromTSP := o.GetOnlineResourceFromTsp(node)
	rs.CPUReservedTSP = resource.NewMilliQuantity(int64(onlineResourceFromTSP[v1.ResourceCPU]), resource.DecimalSI)
	rs.MemoryReservedTSP = resource.NewQuantity(int64(onlineResourceFromTSP[v1.ResourceCPU]), resource.BinarySI)

	o.resourceStatus = rs
	return nil
}

func (o *NodeResourceManager) Name() string {
	return "NodeResourceManager"
}

func (o *NodeResourceManager) getNode() (*v1.Node, error) {
	return o.nodeLister.Get(o.nodeName)
}

func (o *NodeResourceManager) NodeExisted(tsp *predictionapi.TimeSeriesPrediction, addresses []v1.NodeAddress) error {
	address := tsp.Spec.TargetRef.Name
	if address == "" {
		return fmt.Errorf("tsp %s target is not specified", tsp.Name)
	}

	// the reason we use node ip instead of node name as the target name is
	// some monitoring system does not persist node name
	for _, addr := range addresses {
		if addr.Address == address {
			return nil
		}
	}
	return fmt.Errorf("address %s of TSP %s mismatch this node", tsp.Name, address)
}

func (o *NodeResourceManager) updateResource() error {

	origin, err := o.getNode()
	if err != nil {
		return err
	}
	messages := []string{}
	node := origin.DeepCopy()
	updateIfNeed := func(name v1.ResourceName, next resource.Quantity) {
		if existed := node.Status.Capacity[name]; math.Abs(existed.AsApproximateFloat64()-next.AsApproximateFloat64()) >= MinDeltaRatio*existed.AsApproximateFloat64() {
			round := *resource.NewQuantity(next.Value(), next.Format)
			node.Status.Capacity[name] = round
			node.Status.Allocatable[name] = round
			messages = append(messages, fmt.Sprintf("resource %s: %s -> %s (before round: %s)", name.String(), existed.String(), round.String(), next.String()))
		}
	}

	onlineCPU := o.resourceStatus.CPUUsage.DeepCopy()
	onlineCPU.Sub(*o.resourceStatus.CPUUsageOffline)
	onlineCPU.Add(*o.resourceStatus.CPUSetIdle)
	if onlineCPU.Cmp(*o.resourceStatus.CPUReservedTSP) == -1 {
		onlineCPU = o.resourceStatus.CPUReservedTSP.DeepCopy()
	}
	onlineCPU.Add(*o.resourceStatus.CPUReserved)
	// TODO should use allocatable CPU ???
	elasticCPU := node.Status.Allocatable.Cpu().DeepCopy()
	elasticCPU.Sub(onlineCPU)
	updateIfNeed(known.ElasticCPU, elasticCPU)

	onlineMemory := o.resourceStatus.MemoryUsage.DeepCopy()
	onlineMemory.Sub(*o.resourceStatus.MemoryUsageOffline)
	if onlineMemory.Cmp(*o.resourceStatus.MemoryReservedTSP) == -1 {
		onlineMemory = o.resourceStatus.MemoryReservedTSP.DeepCopy()
	}
	onlineMemory.Add(*o.resourceStatus.MemoryReserved)
	// TODO should use allocatable memory?
	elasticMemory := node.Status.Allocatable.Memory().DeepCopy()
	elasticMemory.Sub(onlineMemory)
	updateIfNeed(known.ElasticMemory, elasticMemory)

	if reflect.DeepEqual(node.Status, origin.Status) {
		return nil
	}
	if _, err = o.client.CoreV1().Nodes().UpdateStatus(context.TODO(), node, metav1.UpdateOptions{}); err != nil {
		return err
	}
	o.recorder.Event(node, v1.EventTypeNormal, "UpdateElasticResource", strings.Join(messages, ","))
	return nil
}

// TODO GetOnlineResourceFromTsp should return error when get PredictionResource !!!
func (o *NodeResourceManager) GetOnlineResourceFromTsp(node *v1.Node) map[v1.ResourceName]float64 {
	onlineResource := map[v1.ResourceName]float64{
		v1.ResourceCPU:    0,
		v1.ResourceMemory: 0,
	}

	tsp, err := o.tspLister.TimeSeriesPredictions(known.CraneSystemNamespace).Get(o.tspName)
	if err != nil {
		klog.Errorf("Failed to get tsp: %#v", err)
		return onlineResource
	}

	if err := o.NodeExisted(tsp, node.Status.Addresses); err != nil {
		klog.ErrorS(err, "match tsp and node failed")
		return onlineResource
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
				if onlineResource[resourceName] < nextUsage {
					onlineResource[resourceName] = nextUsage
				}
			}
		}
	}
	return onlineResource
}

func (o *NodeResourceManager) GetResource() *known.ResourceStatus {
	return o.resourceStatus
}

func getReserveResourcePercentFromNodeAnnotations(annotations map[string]string, resourceName string) (float64, bool) {
	if annotations == nil {
		return 0, false
	}
	reserveResourcePercentStr, ok := annotations[fmt.Sprintf(NodeReserveResourcePercentageAnnotationPrefix, resourceName)]
	if !ok {
		return 0, false
	}
	reserveResourcePercent, err := utils.ParsePercentage(reserveResourcePercentStr)
	if err != nil {
		return 0, false
	}
	return reserveResourcePercent, ok
}
