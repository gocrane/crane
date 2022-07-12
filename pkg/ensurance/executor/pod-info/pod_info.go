package pod_info

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
	"github.com/gocrane/crane/pkg/common"
	stypes "github.com/gocrane/crane/pkg/ensurance/collector/types"
	"github.com/gocrane/crane/pkg/utils"
)

type ClassAndPriority struct {
	PodQOSClass        v1.PodQOSClass
	PriorityClassValue int32
}

type PodType string

const (
	ThrottleDown PodType = "ThrottleDown"
	ThrottleUp   PodType = "ThrottleUp"
	Evict        PodType = "Evict"
)

type ContainerUsage struct {
	ContainerName string
	ContainerId   string
	Value         float64
}

func GetUsageById(usages []ContainerUsage, containerId string) (ContainerUsage, error) {
	for _, v := range usages {
		if v.ContainerId == containerId {
			return v, nil
		}
	}

	return ContainerUsage{}, fmt.Errorf("containerUsage not found")
}

func GetPodUsage(metricName string, stateMap map[string][]common.TimeSeries, pod *v1.Pod) (float64, []ContainerUsage) {
	var podUsage = 0.0
	var containerUsages []ContainerUsage
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
				containerUsages = append(containerUsages, ContainerUsage{ContainerId: labelMaps[common.LabelNameContainerId],
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
	PodKey              types.NamespacedName
	ClassAndPriority    ClassAndPriority
	PodCPUUsage         float64
	ContainerCPUUsages  []ContainerUsage
	PodCPUShare         float64
	ContainerCPUShares  []ContainerUsage
	PodCPUQuota         float64
	ContainerCPUQuotas  []ContainerUsage
	PodCPUPeriod        float64
	ContainerCPUPeriods []ContainerUsage
	ExtCpuBeUsed        bool
	ExtCpuLimit         int64
	ExtCpuRequest       int64
	StartTime           *metav1.Time

	PodType PodType

	CPUThrottle    CPURatio
	MemoryThrottle MemoryThrottleExecutor

	DeletionGracePeriodSeconds *int32

	HasBeenActioned bool
}

func HasNoExecutedPod(pods []PodContext) bool {
	for _, p := range pods {
		if p.HasBeenActioned == false {
			return true
		}
	}
	return false
}

func GetFirstNoExecutedPod(pods []PodContext) int {
	for index, p := range pods {
		if p.HasBeenActioned == false {
			return index
		}
	}
	return -1
}

func BuildPodBasicInfo(pod *v1.Pod, stateMap map[string][]common.TimeSeries, action *ensuranceapi.AvoidanceAction, podType PodType) PodContext {
	var podContext PodContext

	podContext.ClassAndPriority = ClassAndPriority{PodQOSClass: pod.Status.QOSClass, PriorityClassValue: utils.GetInt32withDefault(pod.Spec.Priority, 0)}
	podContext.PodKey = types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}

	podContext.PodCPUUsage, podContext.ContainerCPUUsages = GetPodUsage(string(stypes.MetricNameContainerCpuTotalUsage), stateMap, pod)
	podContext.PodCPUShare, podContext.ContainerCPUShares = GetPodUsage(string(stypes.MetricNameContainerCpuLimit), stateMap, pod)
	podContext.PodCPUQuota, podContext.ContainerCPUQuotas = GetPodUsage(string(stypes.MetricNameContainerCpuQuota), stateMap, pod)
	podContext.PodCPUPeriod, podContext.ContainerCPUPeriods = GetPodUsage(string(stypes.MetricNameContainerCpuPeriod), stateMap, pod)
	podContext.ExtCpuBeUsed, podContext.ExtCpuLimit, podContext.ExtCpuRequest = utils.ExtResourceAllocated(pod, v1.ResourceCPU)
	podContext.StartTime = pod.Status.StartTime

	if action.Spec.Throttle != nil {
		podContext.CPUThrottle.MinCPURatio = uint64(action.Spec.Throttle.CPUThrottle.MinCPURatio)
		podContext.CPUThrottle.StepCPURatio = uint64(action.Spec.Throttle.CPUThrottle.StepCPURatio)
	}

	podContext.PodType = podType

	return podContext
}

func CompareClassAndPriority(a, b ClassAndPriority) int32 {
	qosClassCmp := comparePodQosClass(a.PodQOSClass, b.PodQOSClass)
	if qosClassCmp != 0 {
		return qosClassCmp
	}
	if a.PriorityClassValue == b.PriorityClassValue {
		return 0
	} else if a.PriorityClassValue < b.PriorityClassValue {
		return -1
	}
	return 1
}

func (s ClassAndPriority) Less(i ClassAndPriority) bool {
	if comparePodQosClass(s.PodQOSClass, i.PodQOSClass) == 1 {
		return false
	}

	if comparePodQosClass(s.PodQOSClass, i.PodQOSClass) == -1 {
		return true
	}

	return s.PriorityClassValue < i.PriorityClassValue
}

func (s ClassAndPriority) Greater(i ClassAndPriority) bool {
	if comparePodQosClass(s.PodQOSClass, i.PodQOSClass) == 1 {
		return true
	}

	if comparePodQosClass(s.PodQOSClass, i.PodQOSClass) == -1 {
		return false
	}

	return s.PriorityClassValue > i.PriorityClassValue
}

func GetMaxQOSPriority(podLister corelisters.PodLister, podTypes []types.NamespacedName) (types.NamespacedName, ClassAndPriority) {

	var podType types.NamespacedName
	var scheduledQOSPriority ClassAndPriority

	for _, podNamespace := range podTypes {
		if pod, err := podLister.Pods(podNamespace.Namespace).Get(podNamespace.Name); err != nil {
			klog.V(6).Infof("Warning: getMaxQOSPriority get pod %s not found", podNamespace.String())
			continue
		} else {
			var priority = ClassAndPriority{PodQOSClass: pod.Status.QOSClass, PriorityClassValue: utils.GetInt32withDefault(pod.Spec.Priority, 0) - 1}
			if priority.Greater(scheduledQOSPriority) {
				scheduledQOSPriority = priority
				podType = podNamespace
			}
		}
	}

	return podType, scheduledQOSPriority
}

// We defined guaranteed is the highest qos class, burstable is the middle level
// bestEffort is the lowest
// if a qos class is greater than b, return 1
// if a qos class is less than b, return -1
// if a qos class equal with b , return 0
func comparePodQosClass(a v1.PodQOSClass, b v1.PodQOSClass) int32 {
	switch b {
	case v1.PodQOSGuaranteed:
		if a == v1.PodQOSGuaranteed {
			return 0
		} else {
			return -1
		}
	case v1.PodQOSBurstable:
		if a == v1.PodQOSGuaranteed {
			return 1
		} else if a == v1.PodQOSBurstable {
			return 0
		} else {
			return -1
		}
	case v1.PodQOSBestEffort:
		if (a == v1.PodQOSGuaranteed) || (a == v1.PodQOSBurstable) {
			return 1
		} else if a == v1.PodQOSBestEffort {
			return 0
		} else {
			return -1
		}
	default:
		if (a == v1.PodQOSGuaranteed) || (a == v1.PodQOSBurstable) || (a == v1.PodQOSBestEffort) {
			return 1
		} else {
			return 0
		}
	}
}
