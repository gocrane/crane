package podinfo

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
	"github.com/gocrane/crane/pkg/common"
	stypes "github.com/gocrane/crane/pkg/ensurance/collector/types"
	"github.com/gocrane/crane/pkg/utils"
)

type ClassAndPriority struct {
	PodQOSClass        v1.PodQOSClass
	PriorityClassValue int32
}

type ActionType string

const (
	ThrottleDown ActionType = "ThrottleDown"
	ThrottleUp   ActionType = "ThrottleUp"
	Evict        ActionType = "Evict"
)

type ContainerState struct {
	ContainerName string
	ContainerId   string
	Value         float64
}

func GetUsageById(usages []ContainerState, containerId string) (ContainerState, error) {
	for _, v := range usages {
		if v.ContainerId == containerId {
			return v, nil
		}
	}

	return ContainerState{}, fmt.Errorf("containerUsage not found")
}

func GetPodUsage(metricName string, stateMap map[string][]common.TimeSeries, pod *v1.Pod) (float64, []ContainerState) {
	var podUsage = 0.0
	var containerUsages []ContainerState
	var podMaps = map[string]string{common.LabelNamePodName: pod.Name, common.LabelNamePodNamespace: pod.Namespace, common.LabelNamePodUid: string(pod.UID)}
	state, ok := stateMap[metricName]
	if !ok {
		return podUsage, containerUsages
	}
	for _, vv := range state {
		var labelMaps = common.Labels2Maps(vv.Labels)
		if utils.ContainMaps(labelMaps, podMaps) {
			if labelMaps[common.LabelNameContainerId] == "" {
				podUsage = vv.Samples[0].Value
			} else {
				containerUsages = append(containerUsages, ContainerState{ContainerId: labelMaps[common.LabelNameContainerId],
					ContainerName: labelMaps[common.LabelNameContainerName], Value: vv.Samples[0].Value})
			}
		}
	}

	return podUsage, containerUsages
}

type CPURatio struct {
	//the min of cpu ratio for pods
	MinCPURatio uint64 `json:"minCPURatio,omitempty"`

	//the step of cpu share and limit for once down-size (1-100)
	StepCPURatio uint64 `json:"stepCPURatio,omitempty"`
}

type MemoryThrottleExecutor struct {
	// to force gc the page cache of low level pods
	ForceGC bool `json:"forceGC,omitempty"`
}

type PodContext struct {
	Key                        types.NamespacedName
	QOSClass                   v1.PodQOSClass
	Priority                   int32
	StartTime                  *metav1.Time
	DeletionGracePeriodSeconds *int32

	ElasticCPU                                                                      int64
	PodCPUUsage, PodCPUShare, PodCPUQuota, PodCPUPeriod                             float64
	ContainerCPUUsages, ContainerCPUShares, ContainerCPUQuotas, ContainerCPUPeriods []ContainerState

	ActionType  ActionType
	CPUThrottle CPURatio
	Executed    bool
}

func ContainsPendingPod(pods []PodContext) bool {
	for _, p := range pods {
		if p.Executed == false {
			return true
		}
	}
	return false
}

func GetFirstPendingPod(pods []PodContext) int {
	for index, p := range pods {
		if p.Executed == false {
			return index
		}
	}
	return -1
}

func BuildPodActionContext(pod *v1.Pod, stateMap map[string][]common.TimeSeries, action *ensuranceapi.AvoidanceAction, actionType ActionType) PodContext {
	var podContext PodContext

	podContext.QOSClass = pod.Status.QOSClass
	podContext.Priority = utils.GetInt32withDefault(pod.Spec.Priority, 0)

	podContext.Key = types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}

	podContext.PodCPUUsage, podContext.ContainerCPUUsages = GetPodUsage(string(stypes.MetricNameContainerCpuTotalUsage), stateMap, pod)
	podContext.PodCPUShare, podContext.ContainerCPUShares = GetPodUsage(string(stypes.MetricNameContainerCpuLimit), stateMap, pod)
	podContext.PodCPUQuota, podContext.ContainerCPUQuotas = GetPodUsage(string(stypes.MetricNameContainerCpuQuota), stateMap, pod)
	podContext.PodCPUPeriod, podContext.ContainerCPUPeriods = GetPodUsage(string(stypes.MetricNameContainerCpuPeriod), stateMap, pod)
	podContext.ElasticCPU = utils.GetElasticResourceLimit(pod, v1.ResourceCPU)
	podContext.StartTime = pod.Status.StartTime

	if action.Spec.Throttle != nil {
		podContext.CPUThrottle.MinCPURatio = uint64(action.Spec.Throttle.CPUThrottle.MinCPURatio)
		podContext.CPUThrottle.StepCPURatio = uint64(action.Spec.Throttle.CPUThrottle.StepCPURatio)
	}

	podContext.ActionType = actionType

	return podContext
}
