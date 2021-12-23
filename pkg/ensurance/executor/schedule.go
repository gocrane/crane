package executor

import (
	v1 "k8s.io/api/core/v1"

	client "github.com/gocrane/crane/pkg/ensurance/client"
	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/log"
)

const (
	DefaultCoolDownSeconds = 300
)

type ScheduledExecutor struct {
	DisableScheduledQOSPriority *ScheduledQOSPriority
	RestoreScheduledQOSPriority *ScheduledQOSPriority
}

type ScheduledQOSPriority struct {
	PodQOSClass        v1.PodQOSClass
	PriorityClassValue int32
}

func (b *ScheduledExecutor) Avoid(ctx *ExecuteContext) error {
	log.Logger().V(4).Info("Avoid", "DisableScheduledExecutor", *b)

	if b.DisableScheduledQOSPriority == nil {
		return nil
	}

	node, err := ctx.NodeLister.Get(ctx.NodeName)
	if err != nil {
		return err
	}

	// update node condition for block scheduled
	if updateNode, needUpdate := client.UpdateNodeConditions(node, v1.NodeCondition{Type: known.EnsuranceAnalyzedPressureConditionKey, Status: v1.ConditionTrue}); needUpdate {
		if err := client.UpdateNodeStatus(ctx.Client, updateNode, nil); err != nil {
			return err
		}
	}

	// update node taint for block scheduled
	if updateNode, needUpdate := client.UpdateNodeTaints(node, v1.Taint{Key: known.EnsuranceAnalyzedPressureTaintKey, Effect: v1.TaintEffectPreferNoSchedule}); needUpdate {
		if err := client.UpdateNode(ctx.Client, updateNode, nil); err != nil {
			return err
		}
	}

	return nil
}

func (b *ScheduledExecutor) Restore(ctx *ExecuteContext) error {
	log.Logger().V(4).Info("Restore", "DisableScheduledExecutor", *b)

	if b.RestoreScheduledQOSPriority == nil {
		return nil
	}

	node, err := ctx.NodeLister.Get(ctx.NodeName)
	if err != nil {
		return err
	}

	// update node condition for restored scheduled
	if updateNode, needUpdate := client.UpdateNodeConditions(node, v1.NodeCondition{Type: known.EnsuranceAnalyzedPressureConditionKey, Status: v1.ConditionFalse}); needUpdate {
		if err := client.UpdateNodeStatus(ctx.Client, updateNode, nil); err != nil {
			return err
		}
	}

	// update node taint for restored scheduled
	if updateNode, needUpdate := client.RemoveNodeTaints(node, v1.Taint{Key: known.EnsuranceAnalyzedPressureTaintKey, Effect: v1.TaintEffectPreferNoSchedule}); needUpdate {
		log.Logger().V(4).Info("RemoveNodeTaints update true")
		if err := client.UpdateNode(ctx.Client, updateNode, nil); err != nil {
			return err
		}
	} else {
		log.Logger().V(4).Info("RemoveNodeTaints update false")
	}

	return nil
}

func (s ScheduledQOSPriority) Less(i ScheduledQOSPriority) bool {
	if comparePodQos(s.PodQOSClass, i.PodQOSClass) == 1 {
		return false
	}

	if comparePodQos(s.PodQOSClass, i.PodQOSClass) == -1 {
		return true
	}

	return s.PriorityClassValue < i.PriorityClassValue
}

func (s ScheduledQOSPriority) Greater(i ScheduledQOSPriority) bool {
	if comparePodQos(s.PodQOSClass, i.PodQOSClass) == 1 {
		return true
	}

	if comparePodQos(s.PodQOSClass, i.PodQOSClass) == -1 {
		return false
	}

	return s.PriorityClassValue > i.PriorityClassValue
}

// We defined guaranteed is the highest qos class, burstable is the middle level
// bestEffort is the lowest
// if a qos class is greater than b, return 1
// if a qos class is less than b, return -1
// if a qos class equal with b , return 0
func comparePodQos(a v1.PodQOSClass, b v1.PodQOSClass) int32 {
	switch a {
	case v1.PodQOSGuaranteed:
		if b == v1.PodQOSGuaranteed {
			return 0
		} else {
			return -1
		}
	case v1.PodQOSBurstable:
		if b == v1.PodQOSGuaranteed {
			return 1
		} else if b == v1.PodQOSBurstable {
			return 0
		} else {
			return -1
		}
	case v1.PodQOSBestEffort:
		if (b == v1.PodQOSGuaranteed) || (b == v1.PodQOSBurstable) {
			return 1
		} else if b == v1.PodQOSBestEffort {
			return 0
		} else {
			return -1
		}
	default:
		if (b == v1.PodQOSGuaranteed) || (b == v1.PodQOSBurstable) || (b == v1.PodQOSBestEffort) {
			return 1
		} else {
			return 0
		}
	}
}
