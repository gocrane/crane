package executor

import (
	v1 "k8s.io/api/core/v1"

	einformer "github.com/gocrane/crane/pkg/ensurance/informer"
	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/utils/log"
)

type BlockScheduledExecutor struct {
	BlockScheduledQOSPriority   *ScheduledQOSPriority
	RestoreScheduledQOSPriority *ScheduledQOSPriority
}

type ScheduledQOSPriority struct {
	PodQOSClass        v1.PodQOSClass
	PriorityClassValue int32
}

func (b *BlockScheduledExecutor) Avoid(ctx *ExecuteContext) error {
	log.Logger().V(4).Info("Avoid", "BlockScheduledExecutor", *b)

	if b.BlockScheduledQOSPriority == nil {
		return nil
	}

	node, err := einformer.GetNodeFromInformer(ctx.NodeInformer, ctx.NodeName)
	if err != nil {
		return err
	}

	// update node condition for block scheduled
	if updateNode, needUpdate := einformer.UpdateNodeConditions(node, v1.NodeCondition{Type: known.EnsuranceAnalyzedPressureConditionKey, Status: v1.ConditionTrue}); needUpdate {
		if err := einformer.UpdateNodeStatus(ctx.Client, updateNode, nil); err != nil {
			return err
		}
	}

	// update node taint for block scheduled
	if updateNode, needUpdate := einformer.UpdateNodeTaints(node, v1.Taint{Key: known.EnsuranceAnalyzedPressureTaintKey, Effect: v1.TaintEffectPreferNoSchedule}); needUpdate {
		if err := einformer.UpdateNode(ctx.Client, updateNode, nil); err != nil {
			return err
		}
	}

	return nil
}

func (b *BlockScheduledExecutor) Restore(ctx *ExecuteContext) error {
	log.Logger().V(4).Info("Restore", "BlockScheduledExecutor", *b)

	if b.RestoreScheduledQOSPriority == nil {
		return nil
	}

	node, err := einformer.GetNodeFromInformer(ctx.NodeInformer, ctx.NodeName)
	if err != nil {
		return err
	}

	// update node condition for restored scheduled
	if updateNode, needUpdate := einformer.UpdateNodeConditions(node, v1.NodeCondition{Type: known.EnsuranceAnalyzedPressureConditionKey, Status: v1.ConditionFalse}); needUpdate {
		if err := einformer.UpdateNodeStatus(ctx.Client, updateNode, nil); err != nil {
			return err
		}
	}

	// update node taint for restored scheduled
	if updateNode, needUpdate := einformer.RemoveNodeTaints(node, v1.Taint{Key: known.EnsuranceAnalyzedPressureTaintKey, Effect: v1.TaintEffectPreferNoSchedule}); needUpdate {
		log.Logger().V(4).Info("RemoveNodeTaints update true")
		if err := einformer.UpdateNode(ctx.Client, updateNode, nil); err != nil {
			return err
		}
	} else {
		log.Logger().V(4).Info("RemoveNodeTaints update false")
	}

	return nil
}

func (s ScheduledQOSPriority) Less(i ScheduledQOSPriority) bool {

	if getPodQosLevel(s.PodQOSClass) < getPodQosLevel(i.PodQOSClass) {
		return true
	}

	if getPodQosLevel(s.PodQOSClass) > getPodQosLevel(i.PodQOSClass) {
		return false
	}

	return s.PriorityClassValue < i.PriorityClassValue
}

func (s ScheduledQOSPriority) Greater(i ScheduledQOSPriority) bool {

	if getPodQosLevel(s.PodQOSClass) < getPodQosLevel(i.PodQOSClass) {
		return false
	}

	if getPodQosLevel(s.PodQOSClass) > getPodQosLevel(i.PodQOSClass) {
		return true
	}

	return s.PriorityClassValue > i.PriorityClassValue
}

func getPodQosLevel(a v1.PodQOSClass) uint64 {
	switch a {
	case v1.PodQOSGuaranteed:
		return 3
	case v1.PodQOSBurstable:
		return 2
	case v1.PodQOSBestEffort:
		return 1
	default:
		return 0
	}
}
