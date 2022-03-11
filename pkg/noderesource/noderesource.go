package noderesource

import (
	"context"
	"encoding/json"
	"fmt"
	craneclientset "github.com/gocrane/api/pkg/generated/clientset/versioned"
	predictionv1alpha1 "github.com/gocrane/api/pkg/generated/informers/externalversions/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/utils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"math"
	"strings"
)

const (
	ExtResourcePrefix                             = "ext-resource.node.gocrane.io/%s"
	MinDeltaRatio                                 = 0.1
	NodeReserveResourcePercentageAnnotationPrefix = "reserve.node.gocrane.io/%s"
)

type newCollectorFunc func(context *CollectContext) (Collector, error)

var nodeResourceFunc = make(map[string]newCollectorFunc, 10)

func registerMetrics(collectorName string, newCollector newCollectorFunc) {
	nodeResourceFunc[collectorName] = newCollector
}

// Resource is the cpu and memory configuration
type Resource struct {
	CpuPercent *float64
	MemPercent *float64
}

type MetricTimeSeries struct {
	DataSourceName string
	TimeSeriesList []common.TimeSeries
}

type NodeResource struct {
	stateChan chan struct {
		stateMap      map[string][]MetricTimeSeries
		collectorName string
	}
	Client                     *kubernetes.Clientset
	CraneClient                *craneclientset.Clientset
	nodeName                   string
	timeSeriesPredictionSynced cache.InformerSynced
	nodeLister                 corelisters.NodeLister
	nodeSynced                 cache.InformerSynced
	recorder                   record.EventRecorder
	reserveResource            Resource
	collectorList              []Collector
}

func NewNodeResource(
	nodeName string,
	kubeClient *kubernetes.Clientset,
	craneClient *craneclientset.Clientset,
	nodeInformer coreinformers.NodeInformer,
	timeSeriesPredictionInformer predictionv1alpha1.TimeSeriesPredictionInformer,
	reserveCpuPercentStr string,
	reserveMemoryPercentStr string,
	collectorNames []string,
	cpuStateProvider *utils.CpuStateProvider,
	tspName string,
) *NodeResource {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "crane-agent"})
	stateChan := make(chan struct {
		stateMap      map[string][]MetricTimeSeries
		collectorName string
	})

	reserveCpuPercent, _ := utils.ParsePercentage(reserveCpuPercentStr)
	reserveMemoryPercent, _ := utils.ParsePercentage(reserveMemoryPercentStr)

	collectContext := CollectContext{
		TimeSeriesPredictionInformer: timeSeriesPredictionInformer,
		Recorder:                     recorder,
		NodeName:                     nodeName,
		CpuStateProvider:             cpuStateProvider,
		TspName:                      tspName,
	}

	collectorList := make([]Collector, 0)

	klog.Infof("collectorNames: %s", strings.Join(collectorNames, ","))

	for _, collectorName := range collectorNames {
		if f, ok := nodeResourceFunc[collectorName]; ok {
			if c, err := f(&collectContext); err == nil {
				collectorList = append(collectorList, c)
			}
		}
	}

	return &NodeResource{
		stateChan:                  stateChan,
		Client:                     kubeClient,
		CraneClient:                craneClient,
		nodeName:                   nodeName,
		nodeLister:                 nodeInformer.Lister(),
		nodeSynced:                 nodeInformer.Informer().HasSynced,
		timeSeriesPredictionSynced: timeSeriesPredictionInformer.Informer().HasSynced,
		recorder:                   recorder,
		reserveResource: Resource{
			CpuPercent: &reserveCpuPercent,
			MemPercent: &reserveMemoryPercent,
		},
		collectorList: collectorList,
	}
}

func (nr *NodeResource) Run(stop <-chan struct{}) {
	klog.Infof("Starting noderesource analyzer.")

	// Wait for the caches to be synced before starting workers
	if !cache.WaitForNamedCacheSync("noderesource-analyzer",
		stop,
		nr.nodeSynced,
		nr.timeSeriesPredictionSynced,
	) {
		return
	}
	for _, c := range nr.collectorList {
		c.Run(stop, nr.stateChan)
	}
	go func() {
		for {
			select {
			case stateSingle := <-nr.stateChan:
				klog.V(4).Infof("nodeResource got stateSingle from %s, state: %v", stateSingle.collectorName, stateSingle.stateMap)
				state := nr.GenerateFullyAnalyzeData(stateSingle.stateMap, stateSingle.collectorName)
				nr.Analyze(state)
			case <-stop:
				klog.Infof("NodeResource exit")
				return
			}
		}
	}()
}

func (nr *NodeResource) GenerateFullyAnalyzeData(state map[string][]MetricTimeSeries, collectorName string) map[string][]MetricTimeSeries {
	for _, c := range nr.collectorList {
		if c.Name() == collectorName {
			continue
		}
		tmpState := c.GetLastState()
		for resourceName := range tmpState {
			if _, ok := state[resourceName]; ok {
				state[resourceName] = append(state[resourceName], tmpState[resourceName]...)
			} else {
				state[resourceName] = tmpState[resourceName]
			}
		}
	}
	return state
}

func (nr *NodeResource) Analyze(state map[string][]MetricTimeSeries) {
	klog.V(4).Infof("state of NodeResource: %v", state)
	node, err := nr.nodeLister.Get(nr.nodeName)
	if err != nil {
		klog.Errorf("Failed to get node: %v", err)
		return
	}
	nodeCopy := node.DeepCopy()
	effectDataSourceNameMap := make(map[string]string)
	lowerLimitResourceNameMap := make(map[string]float64)
	//获取指标上限值
	for resourceName, mTsList := range state {
		for _, metricTimeSeries := range mTsList {
			if metricTimeSeries.DataSourceName != GetRealtimeCollectorName() {
				continue
			}
			for _, timeSeries := range metricTimeSeries.TimeSeriesList {
				for _, sample := range timeSeries.Samples {
					if lowerLimitResourceNameMap[resourceName] < sample.Value*0.9 {
						lowerLimitResourceNameMap[resourceName] = sample.Value * 0.9
					}
				}
			}
		}
	}
	for resourceName, mTsList := range state {
		var miniIdle, nextIdle *float64
		for _, metricTimeSeries := range mTsList {
			for _, timeSeries := range metricTimeSeries.TimeSeriesList {
				for _, sample := range timeSeries.Samples {
					if sample.Timestamp == 0 {
						continue
					}
					nextIdle = &sample.Value
					if lowerLimitResource, ok := lowerLimitResourceNameMap[resourceName]; ok {
						if *nextIdle < lowerLimitResource {
							klog.V(4).Infof("The predicted sample of resource %s is lower than the lowerLimitValue, sampleValue: %f, lowerLimitValue: %f", resourceName, nextIdle, lowerLimitResource)
							continue
						}
					}
					if miniIdle == nil {
						miniIdle = nextIdle
						effectDataSourceNameMap[resourceName] = metricTimeSeries.DataSourceName
					}
					if *miniIdle > *nextIdle {
						miniIdle = nextIdle
						effectDataSourceNameMap[resourceName] = metricTimeSeries.DataSourceName
					}
				}
			}
		}
		nr.BuildNodeStatus(resourceName, *miniIdle, nodeCopy)
	}

	if !equality.Semantic.DeepEqual(&node.Status, &nodeCopy.Status) {
		var jsonPatch []byte
		effectDataSources, _ := json.Marshal(effectDataSourceNameMap)
		nr.recorder.Event(utils.GetNodeRef(nr.nodeName), v1.EventTypeNormal, "RecommendExtendResource", fmt.Sprintf("Recommend Node Extend Resource Success: %s", effectDataSources))
		oldJson, err := json.Marshal(node)
		newJson, err := json.Marshal(nodeCopy)
		if err == nil {
			jsonPatch, err = strategicpatch.CreateTwoWayMergePatch(oldJson, newJson, nodeCopy)
			klog.V(4).Infof("jsonPatch: %s", jsonPatch)
		}
		if err != nil {
			// update Node status extend-resource info directly
			if _, err := nr.Client.CoreV1().Nodes().UpdateStatus(context.TODO(), nodeCopy, metav1.UpdateOptions{}); err != nil {
				nr.recorder.Event(utils.GetNodeRef(nr.nodeName), v1.EventTypeNormal, "FailedUpdateNodeExtendResource", err.Error())
				klog.Errorf("Failed to update node %s's status extend-resource, %v", nodeCopy.Name, err)
				return
			}
		} else {
			// patch Node status extend-resource info
			if _, err := nr.Client.CoreV1().Nodes().PatchStatus(context.TODO(), nodeCopy.Name, jsonPatch); err != nil {
				nr.recorder.Event(utils.GetNodeRef(nr.nodeName), v1.EventTypeNormal, "FailedPatchNodeExtendResource", err.Error())
				klog.Errorf("Failed to patch node %s's status extend-resource, %v", nodeCopy.Name, err)
				return
			}
		}
		nr.recorder.Event(utils.GetNodeRef(nr.nodeName), v1.EventTypeNormal, "UpdateNode", fmt.Sprintf("Update Node Extend Resource Success"))
	}
}

func (nr *NodeResource) BuildNodeStatus(resourceName string, value float64, node *v1.Node) {
	reserveCpuPercent := nr.reserveResource.CpuPercent
	//reserveMemoryPercent := nr.reserveResource.MemPercent

	if nodeReserveCpuPercent, ok := getReserveResourcePercentFromNodeAnnotations(node.GetAnnotations(), v1.ResourceCPU.String()); ok {
		reserveCpuPercent = &nodeReserveCpuPercent
	}

	//if nodeReserveMemoryPercent, ok := getReserveResourcePercentFromNodeAnnotations(node.GetAnnotations(), v1.ResourceMemory.String()); ok {
	//	reserveMemoryPercent = &nodeReserveMemoryPercent
	//}

	var nextRecommendation float64
	switch resourceName {
	case v1.ResourceCPU.String():
		// cpu need to be scaled to m as ext resource cannot be decimal
		if reserveCpuPercent != nil {
			nextRecommendation = (value - float64(node.Status.Allocatable.Cpu().Value())**reserveCpuPercent) * 1000
		} else {
			nextRecommendation = value * 1000
		}
	//case v1.ResourceMemory.String():
	//	// unit of memory in prometheus is in Ki, need to be converted to byte
	//	if reserveMemoryPercent != nil {
	//		nextRecommendation = float64(node.Status.Allocatable.Memory().Value()) - float64(node.Status.Allocatable.Memory().Value())**reserveMemoryPercent - (value * 1000)
	//	} else {
	//		nextRecommendation = float64(node.Status.Allocatable.Memory().Value()) - (value * 1000)
	//	}
	default:
	}
	if nextRecommendation < 0 {
		klog.V(4).Infof("Unexpected recommendation,nodeName %s, value %v, nextRecommendation %v", node.Name, value, nextRecommendation)
		nextRecommendation = 0
	}
	extResourceName := fmt.Sprintf(ExtResourcePrefix, resourceName)
	resValue, exists := node.Status.Capacity[v1.ResourceName(extResourceName)]
	if exists && resValue.Value() != 0 &&
		math.Abs(float64(resValue.Value())-
			nextRecommendation)/float64(resValue.Value()) <= MinDeltaRatio {
		return
	}
	node.Status.Capacity[v1.ResourceName(extResourceName)] =
		*resource.NewQuantity(int64(nextRecommendation), resource.DecimalSI)
	node.Status.Allocatable[v1.ResourceName(extResourceName)] =
		*resource.NewQuantity(int64(nextRecommendation), resource.DecimalSI)
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

func (nr *NodeResource) Name() string {
	return "NodeResourceManager"
}
