package executor

import v1 "k8s.io/api/core/v1"

type BlockScheduledExecutor struct {
	BlockScheduledQOSPriority   *ScheduledQOSPriority
	RestoreScheduledQOSPriority *ScheduledQOSPriority
}

type ScheduledQOSPriority struct {
	PodQOSClass        v1.PodQOSClass
	PriorityClassValue uint64
}

func (b *BlockScheduledExecutor) Avoid(ctx *ExecuteContext) error {
	// update node condition for block scheduled
	if b.BlockScheduledQOSPriority != nil {
		// einformer.updateNodeConditions
		// einformer.updateNodeStatus
	}
	return nil
}

func (b *BlockScheduledExecutor) Restore(ctx *ExecuteContext) error {
	// update node condition for restored scheduled
	if b.RestoreScheduledQOSPriority != nil {
		// einformer.updateNodeConditions
		// einformer.updateNodeStatus
	}
	return nil
}
