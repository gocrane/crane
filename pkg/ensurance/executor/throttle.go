package executor

import (
	"fmt"
	"github.com/gocrane/crane/pkg/ensurance/client"
	"github.com/gocrane/crane/pkg/log"
	"k8s.io/apimachinery/pkg/types"
)

type ThrottleExecutor struct {
	ThrottlePods ThrottlePods
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

type CPUThrottleExecutor struct {
	CPUDownAction CPURatio
	CPUUpAction   CPURatio
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
	CPUThrottle        CPUThrottleExecutor
	MemoryThrottle     MemoryThrottleExecutor
	PodTypes           types.NamespacedName
	PodCPUUsage        float64
	ContainerCPUUsages []ContainerUsage
	PodQOSPriority     ScheduledQOSPriority
}

type ContainerUsage struct {
	ContainerName string
	ContainerId   string
	Value         float64
}

func (t *ThrottleExecutor) Avoid(ctx *ExecuteContext) error {

	var bSucceed = true
	var errPodKeys []string

	for _, throttlePod := range t.ThrottlePods {

		pod, err := ctx.PodLister.Pods(throttlePod.PodTypes.Namespace).Get(throttlePod.PodTypes.Name)
		if err != nil {
			bSucceed = false
			errPodKeys = append(errPodKeys, "not found ", throttlePod.PodTypes.String())
			continue
		}

		log.Logger().V(4).Info(fmt.Sprintf("ThrottleExecutor avoid pod %s", log.GenerateObj(pod)))

		ctx.RuntimeClient.ContainerStats()

		err = client.EvictPodWithGracePeriod(ctx.Client, pod, throttlePod.DeletionGracePeriodSeconds)
		if err != nil {
			bSucceed = false
			errPodKeys = append(errPodKeys, "evict failed ", evictPod.PodTypes.String())
			log.Logger().V(4).Info(fmt.Sprintf("Warning: evict failed %s, err %s", evictPod.PodTypes.String(), err.Error()))
			continue
		}

	}

	return nil
}

func (t *ThrottleExecutor) Restore(ctx *ExecuteContext) error {
	return nil
}
