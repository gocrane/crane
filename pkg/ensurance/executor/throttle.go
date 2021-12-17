package executor

import "k8s.io/apimachinery/pkg/types"

type ThrottleExecutor struct {
	CPUThrottle    *CPUThrottle
	MemoryThrottle *MemoryThrottle
	ThrottlePods   []types.NamespacedName
}

type CPUThrottle struct {
	CPUDownAction *CPURatio
	CPUUpAction   *CPURatio
}

type CPURatio struct {
	//the min of cpu ratio for pods
	MinCPURatio uint64 `json:"minCPURatio,omitempty"`

	//the step of cpu share and limit for once down-size (1-100)
	StepCPURatio uint64 `json:"stepCPURatio,omitempty"`
}

type MemoryThrottle struct {
	// to force gc the page cache of low level pods
	ForceGC bool `json:"forceGC,omitempty"`
}

func (t *ThrottleExecutor) Avoid(ctx *ExecuteContext) error {
	return nil
}

func (t *ThrottleExecutor) Restore(ctx *ExecuteContext) error {
	return nil
}
