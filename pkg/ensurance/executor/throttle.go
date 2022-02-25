package executor

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
	cruntime "github.com/gocrane/crane/pkg/ensurance/runtime"
	"github.com/gocrane/crane/pkg/utils"
)

const (
	MAX_UP_QUOTA = 60 * 1000 // 60CU
)

type ThrottleExecutor struct {
	ThrottleDownPods ThrottlePods
	ThrottleUpPods   ThrottlePods
}

type ThrottlePods []ThrottlePod

func (t ThrottlePods) Len() int      { return len(t) }
func (t ThrottlePods) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t ThrottlePods) Less(i, j int) bool {
	return t[i].PodQOSPriority.Less(t[j].PodQOSPriority)
}

func (t ThrottlePods) Find(podTypes types.NamespacedName) int {
	for i, v := range t {
		if v.PodTypes == podTypes {
			return i
		}
	}

	return -1
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

type ThrottlePod struct {
	CPUThrottle         CPURatio
	MemoryThrottle      MemoryThrottleExecutor
	PodTypes            types.NamespacedName
	PodCPUUsage         float64
	ContainerCPUUsages  []ContainerUsage
	PodCPUShare         float64
	ContainerCPUShares  []ContainerUsage
	PodCPUQuota         float64
	ContainerCPUQuotas  []ContainerUsage
	PodCPUPeriod        float64
	ContainerCPUPeriods []ContainerUsage
	PodQOSPriority      ClassAndPriority
}

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

func (t *ThrottleExecutor) Avoid(ctx *ExecuteContext) error {

	var bSucceed = true
	var errPodKeys []string

	for _, throttlePod := range t.ThrottleDownPods {

		pod, err := ctx.PodLister.Pods(throttlePod.PodTypes.Namespace).Get(throttlePod.PodTypes.Name)
		if err != nil {
			bSucceed = false
			errPodKeys = append(errPodKeys, fmt.Sprintf("pod %s not found", throttlePod.PodTypes.String()))
			continue
		}

		for _, v := range throttlePod.ContainerCPUUsages {
			// pause container to skip
			if v.ContainerName == "" {
				continue
			}

			klog.V(4).Infof("ThrottleExecutor1 avoid container %s/%s", klog.KObj(pod), v.ContainerName)

			containerCPUQuota, err := GetUsageById(throttlePod.ContainerCPUQuotas, v.ContainerId)
			if err != nil {
				bSucceed = false
				errPodKeys = append(errPodKeys, err.Error(), throttlePod.PodTypes.String())
				continue
			}

			containerCPUPeriod, err := GetUsageById(throttlePod.ContainerCPUPeriods, v.ContainerId)
			if err != nil {
				bSucceed = false
				errPodKeys = append(errPodKeys, err.Error(), throttlePod.PodTypes.String())
				continue
			}

			container, err := utils.GetPodContainerByName(pod, v.ContainerName)
			if err != nil {
				bSucceed = false
				errPodKeys = append(errPodKeys, err.Error(), throttlePod.PodTypes.String())
				continue
			}

			var containerCPUQuotaNew float64
			if utils.AlmostEqual(containerCPUQuota.Value, -1.0) || utils.AlmostEqual(containerCPUQuota.Value, 0.0) {
				containerCPUQuotaNew = v.Value * (1.0 - float64(throttlePod.CPUThrottle.StepCPURatio)/100.0)
			} else {
				containerCPUQuotaNew = containerCPUQuota.Value / containerCPUPeriod.Value * (1.0 - float64(throttlePod.CPUThrottle.StepCPURatio)/100.0)
			}

			if requestCPU, ok := container.Resources.Requests[v1.ResourceCPU]; ok {
				if float64(requestCPU.MilliValue())/1000.0 > containerCPUQuotaNew {
					containerCPUQuotaNew = float64(requestCPU.MilliValue()) / 1000.0
				}
			}

			if limitCPU, ok := container.Resources.Limits[v1.ResourceCPU]; ok {
				if float64(limitCPU.MilliValue())/1000.0*float64(throttlePod.CPUThrottle.MinCPURatio)/100 > containerCPUQuotaNew {
					containerCPUQuotaNew = float64(limitCPU.MilliValue()) * float64(throttlePod.CPUThrottle.MinCPURatio) / 1000.0
				}
			}

			klog.V(6).Infof("Prior update container resources containerCPUQuotaNew %.2f, containerCPUQuota.Value %.2f,containerCPUPeriod %.2f",
				containerCPUQuotaNew, containerCPUQuota.Value, containerCPUPeriod.Value)

			if !utils.AlmostEqual(containerCPUQuotaNew*containerCPUPeriod.Value, containerCPUQuota.Value) {
				err = cruntime.UpdateContainerResources(ctx.RuntimeClient, v.ContainerId, cruntime.UpdateOptions{CPUQuota: int64(containerCPUQuotaNew * containerCPUPeriod.Value)})
				if err != nil {
					errPodKeys = append(errPodKeys, fmt.Sprintf("failed to updateResource for %s/%s, error: %v", throttlePod.PodTypes.String(), v.ContainerName, err))
					bSucceed = false
					continue
				} else {
					klog.V(4).Infof("ThrottleExecutor avoid pod %s, container %s, set cpu quota %.2f.",
						klog.KObj(pod), v.ContainerName, containerCPUQuotaNew)
				}
			}
		}
	}

	if !bSucceed {
		return fmt.Errorf("some pod throttle failed,err: %s", strings.Join(errPodKeys, ";"))
	}

	return nil
}

func (t *ThrottleExecutor) Restore(ctx *ExecuteContext) error {
	var bSucceed = true
	var errPodKeys []string

	for _, throttlePod := range t.ThrottleUpPods {

		pod, err := ctx.PodLister.Pods(throttlePod.PodTypes.Namespace).Get(throttlePod.PodTypes.Name)
		if err != nil {
			bSucceed = false
			errPodKeys = append(errPodKeys, "not found ", throttlePod.PodTypes.String())
			continue
		}

		for _, v := range throttlePod.ContainerCPUUsages {

			// pause container to skip
			if v.ContainerName == "" {
				continue
			}

			klog.V(6).Infof("ThrottleExecutor1 restore container %s/%s", klog.KObj(pod), v.ContainerName)

			containerCPUQuota, err := GetUsageById(throttlePod.ContainerCPUQuotas, v.ContainerId)
			if err != nil {
				bSucceed = false
				errPodKeys = append(errPodKeys, err.Error(), throttlePod.PodTypes.String())
				continue
			}

			containerCPUPeriod, err := GetUsageById(throttlePod.ContainerCPUPeriods, v.ContainerId)
			if err != nil {
				bSucceed = false
				errPodKeys = append(errPodKeys, err.Error(), throttlePod.PodTypes.String())
				continue
			}

			container, err := utils.GetPodContainerByName(pod, v.ContainerName)
			if err != nil {
				bSucceed = false
				errPodKeys = append(errPodKeys, err.Error(), throttlePod.PodTypes.String())
				continue
			}

			var containerCPUQuotaNew float64
			if utils.AlmostEqual(containerCPUQuota.Value, -1.0) || utils.AlmostEqual(containerCPUQuota.Value, 0.0) {
				continue
			} else {
				containerCPUQuotaNew = containerCPUQuota.Value / containerCPUPeriod.Value * (1.0 + float64(throttlePod.CPUThrottle.StepCPURatio)/100.0)
			}

			if limitCPU, ok := container.Resources.Limits[v1.ResourceCPU]; ok {
				if float64(limitCPU.MilliValue())/1000.0 < containerCPUQuotaNew {
					containerCPUQuotaNew = float64(limitCPU.MilliValue()) / 1000.0
				}
			} else {
				if containerCPUQuotaNew > MAX_UP_QUOTA*containerCPUPeriod.Value/1000.0 {
					containerCPUQuotaNew = -1
				}
			}

			klog.V(6).Infof("Prior update container resources containerCPUQuotaNew %.2f,containerCPUQuota %.2f,containerCPUPeriod %.2f",
				klog.KObj(pod), containerCPUQuotaNew, containerCPUQuota.Value, containerCPUPeriod.Value)

			if !utils.AlmostEqual(containerCPUQuotaNew*containerCPUPeriod.Value, containerCPUQuota.Value) {

				if utils.AlmostEqual(containerCPUQuotaNew, -1) {
					err = cruntime.UpdateContainerResources(ctx.RuntimeClient, v.ContainerId, cruntime.UpdateOptions{CPUQuota: int64(-1)})
					if err != nil {
						errPodKeys = append(errPodKeys, fmt.Sprintf("Failed to updateResource, err %s", err.Error()), throttlePod.PodTypes.String())
						bSucceed = false
						continue
					}
				} else {
					err = cruntime.UpdateContainerResources(ctx.RuntimeClient, v.ContainerId, cruntime.UpdateOptions{CPUQuota: int64(containerCPUQuotaNew * containerCPUPeriod.Value)})
					if err != nil {
						klog.Errorf("Failed to updateResource, err %s", err.Error())
						errPodKeys = append(errPodKeys, fmt.Sprintf("Failed to updateResource, err %s", err.Error()), throttlePod.PodTypes.String())
						bSucceed = false
						continue
					}
					klog.V(2).Infof("restore kkkkkkkk")
				}
			}
		}
	}

	if !bSucceed {
		return fmt.Errorf("some pod throttle restore failed,err: %s", strings.Join(errPodKeys, ";"))
	}

	return nil
}

func (e *ThrottleExecutor) Deduplicate(throttlePods, throttleUpPods ThrottlePods) {
	for _, t := range throttlePods {
		if i := e.ThrottleDownPods.Find(t.PodTypes); i == -1 {
			e.ThrottleDownPods = append(e.ThrottleDownPods, t)
		} else {
			if t.CPUThrottle.MinCPURatio > e.ThrottleDownPods[i].CPUThrottle.MinCPURatio {
				e.ThrottleDownPods[i].CPUThrottle.MinCPURatio = t.CPUThrottle.MinCPURatio
			}

			if t.CPUThrottle.StepCPURatio > e.ThrottleDownPods[i].CPUThrottle.StepCPURatio {
				e.ThrottleDownPods[i].CPUThrottle.StepCPURatio = t.CPUThrottle.StepCPURatio
			}
		}
	}
	for _, t := range throttleUpPods {

		if i := e.ThrottleUpPods.Find(t.PodTypes); i == -1 {
			e.ThrottleUpPods = append(e.ThrottleUpPods, t)
		} else {
			if t.CPUThrottle.MinCPURatio > e.ThrottleUpPods[i].CPUThrottle.MinCPURatio {
				e.ThrottleUpPods[i].CPUThrottle.MinCPURatio = t.CPUThrottle.MinCPURatio
			}

			if t.CPUThrottle.StepCPURatio > e.ThrottleUpPods[i].CPUThrottle.StepCPURatio {
				e.ThrottleUpPods[i].CPUThrottle.StepCPURatio = t.CPUThrottle.StepCPURatio
			}
		}
	}
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
