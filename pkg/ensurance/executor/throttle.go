package executor

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	cruntime "github.com/gocrane/crane/pkg/ensurance/runtime"
	"github.com/gocrane/crane/pkg/log"
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
	PodQOSPriority      ScheduledQOSPriority
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

		log.Logger().V(2).Info(fmt.Sprintf("throttlePod %+v", throttlePod))

		pod, err := ctx.PodLister.Pods(throttlePod.PodTypes.Namespace).Get(throttlePod.PodTypes.Name)
		if err != nil {
			bSucceed = false
			errPodKeys = append(errPodKeys, "not found ", throttlePod.PodTypes.String())
			continue
		}

		log.Logger().V(2).Info(fmt.Sprintf("ThrottleExecutor1 avoid pod %s", log.GenerateObj(pod)))

		for _, v := range throttlePod.ContainerCPUUsages {
			// pause container to skip
			if v.ContainerName == "" {
				continue
			}

			log.Logger().V(2).Info(fmt.Sprintf("ThrottleExecutor avoid pod %s, container %s.", log.GenerateObj(pod), v.ContainerName))

			log.Logger().Info("aaaaaa")
			containerCPUQuota, err := GetUsageById(throttlePod.ContainerCPUQuotas, v.ContainerId)
			if err != nil {
				bSucceed = false
				errPodKeys = append(errPodKeys, err.Error(), throttlePod.PodTypes.String())
				continue
			}

			log.Logger().Info("bbbbbbb")

			containerCPUPeriod, err := GetUsageById(throttlePod.ContainerCPUPeriods, v.ContainerId)
			if err != nil {
				bSucceed = false
				errPodKeys = append(errPodKeys, err.Error(), throttlePod.PodTypes.String())
				continue
			}

			log.Logger().Info("ccccccc")

			container, err := utils.GetPodContainerByName(pod, v.ContainerName)
			if err != nil {
				bSucceed = false
				errPodKeys = append(errPodKeys, err.Error(), throttlePod.PodTypes.String())
				continue
			}

			log.Logger().Info(fmt.Sprintf("ddddddd Value: %f, containerCPUPeriod.Value: %f, containerCPUQuota %f", v.Value, containerCPUPeriod.Value,
				containerCPUQuota.Value))

			var containerCPUQuotaNew float64
			if utils.AlmostEqual(containerCPUQuota.Value, -1.0) || utils.AlmostEqual(containerCPUQuota.Value, 0.0) {
				containerCPUQuotaNew = v.Value * (1.0 - float64(throttlePod.CPUThrottle.StepCPURatio)/100.0)
			} else {
				containerCPUQuotaNew = containerCPUQuota.Value / containerCPUPeriod.Value * (1.0 - float64(throttlePod.CPUThrottle.StepCPURatio)/100.0)
			}

			log.Logger().Info(fmt.Sprintf("eeeee1 containerCPUQuotaNew %f",
				containerCPUQuotaNew))

			if requestCPU, ok := container.Resources.Requests[v1.ResourceCPU]; ok {
				if float64(requestCPU.Value())/1000.0 > containerCPUQuotaNew {
					containerCPUQuotaNew = float64(requestCPU.Value()) / 1000.0
				}
			}

			log.Logger().Info(fmt.Sprintf("eeeee2 containerCPUQuotaNew %f",
				containerCPUQuotaNew))

			if limitCPU, ok := container.Resources.Limits[v1.ResourceCPU]; ok {
				if float64(limitCPU.Value())/1000.0*float64(throttlePod.CPUThrottle.MinCPURatio) > containerCPUQuotaNew {
					containerCPUQuotaNew = float64(limitCPU.Value()) * float64(throttlePod.CPUThrottle.MinCPURatio) / 1000.0
				}
			}

			log.Logger().Info(fmt.Sprintf("ffffffff containerCPUQuotaNew %f, containerCPUQuota.Value %f",
				containerCPUQuotaNew, containerCPUQuota.Value))

			if !utils.AlmostEqual(containerCPUQuotaNew*containerCPUPeriod.Value, containerCPUQuota.Value) {
				err = cruntime.UpdateContainerResources(ctx.RuntimeClient, v.ContainerId, cruntime.UpdateOptions{CPUQuota: int64(containerCPUQuotaNew * containerCPUPeriod.Value)})
				if err != nil {
					klog.Errorf("Failed to updateResource, err %s", err.Error())
					errPodKeys = append(errPodKeys, fmt.Sprintf("Failed to updateResource, err %s", err.Error()), throttlePod.PodTypes.String())
					bSucceed = false
					continue
				} else {
					log.Logger().V(2).Info(fmt.Sprintf("ThrottleExecutor avoid pod %s, container %s, set cpu quota %.2f.",
						log.GenerateObj(pod), v.ContainerName, containerCPUQuotaNew))
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

		log.Logger().V(4).Info(fmt.Sprintf("ThrottleExecutor restore pod %s", log.GenerateObj(pod)))

		for _, v := range throttlePod.ContainerCPUUsages {

			// pause container to skip
			if v.ContainerName == "" {
				continue
			}

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
				if float64(limitCPU.Value())/1000.0 < containerCPUQuotaNew {
					containerCPUQuotaNew = float64(limitCPU.Value()) / 1000.0
				}
			} else {
				if containerCPUQuotaNew > MAX_UP_QUOTA*containerCPUPeriod.Value/1000.0 {
					containerCPUQuotaNew = -1
				}
			}

			if !utils.AlmostEqual(containerCPUQuotaNew*containerCPUPeriod.Value, containerCPUQuota.Value) {

				if utils.AlmostEqual(containerCPUQuotaNew, -1) {
					err = cruntime.UpdateContainerResources(ctx.RuntimeClient, v.ContainerId, cruntime.UpdateOptions{CPUQuota: int64(-1)})
					if err != nil {
						klog.Errorf("Failed to updateResource to no limited, err %s", err.Error())
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
				}
			}
		}
	}

	if !bSucceed {
		return fmt.Errorf("some pod throttle failed,err: %s", strings.Join(errPodKeys, ";"))
	}

	return nil
}
