package executor

import (
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/metrics"
	"github.com/gocrane/crane/pkg/utils"
)

const (
	DefaultCoolDownSeconds = 300
)

type ScheduleExecutor struct {
	ToBeDisable, ToBeRestore bool
}

func (b *ScheduleExecutor) Avoid(ctx *ExecuteContext) error {
	var start = time.Now()
	metrics.UpdateLastTimeWithSubComponent(string(known.ModuleActionExecutor), string(metrics.SubComponentSchedule), metrics.StepAvoid, start)
	defer metrics.UpdateDurationFromStartWithSubComponent(string(known.ModuleActionExecutor), string(metrics.SubComponentSchedule), metrics.StepAvoid, start)

	klog.V(4).Infof("ScheduleExecutor, ToBeDisable: %v, ToBeRestore: %v", b.ToBeDisable, b.ToBeRestore)

	if !b.ToBeDisable {
		metrics.UpdateExecutorStatus(metrics.SubComponentSchedule, metrics.StepAvoid, 0)
		return nil
	}

	metrics.UpdateExecutorStatus(metrics.SubComponentSchedule, metrics.StepAvoid, 1.0)
	metrics.ExecutorStatusCounterInc(metrics.SubComponentSchedule, metrics.StepAvoid)

	// update node condition for block scheduled
	if _, err := utils.UpdateNodeConditionsStatues(ctx.Client, ctx.NodeLister, ctx.NodeName,
		v1.NodeCondition{Type: known.EnsuranceAnalyzedPressureConditionKey, Status: v1.ConditionTrue}, nil); err != nil {
		return err
	}

	// update node taint for block scheduled
	if _, err := utils.UpdateNodeTaints(ctx.Client, ctx.NodeLister, ctx.NodeName,
		v1.Taint{Key: known.EnsuranceAnalyzedPressureTaintKey, Effect: v1.TaintEffectPreferNoSchedule}, nil); err != nil {
		return err
	}

	return nil
}

func (b *ScheduleExecutor) Restore(ctx *ExecuteContext) error {
	var start = time.Now()
	metrics.UpdateLastTimeWithSubComponent(string(known.ModuleActionExecutor), string(metrics.SubComponentSchedule), metrics.StepRestore, start)
	defer metrics.UpdateDurationFromStartWithSubComponent(string(known.ModuleActionExecutor), string(metrics.SubComponentSchedule), metrics.StepRestore, start)

	klog.V(4).Infof("ScheduleExecutor, ToBeDisable: %v, ToBeRestore: %v", b.ToBeDisable, b.ToBeRestore)

	if !b.ToBeRestore {
		metrics.UpdateExecutorStatus(metrics.SubComponentSchedule, metrics.StepRestore, 0.0)
		return nil
	}

	metrics.UpdateExecutorStatus(metrics.SubComponentSchedule, metrics.StepRestore, 1.0)
	metrics.ExecutorStatusCounterInc(metrics.SubComponentSchedule, metrics.StepRestore)

	// update node condition for restored scheduled
	if _, err := utils.UpdateNodeConditionsStatues(ctx.Client, ctx.NodeLister, ctx.NodeName,
		v1.NodeCondition{Type: known.EnsuranceAnalyzedPressureConditionKey, Status: v1.ConditionFalse}, nil); err != nil {
		return err
	}

	// update node taint for restored scheduled
	if _, err := utils.RemoveNodeTaints(ctx.Client, ctx.NodeLister, ctx.NodeName,
		v1.Taint{Key: known.EnsuranceAnalyzedPressureTaintKey, Effect: v1.TaintEffectPreferNoSchedule}, nil); err != nil {
		return err
	}

	return nil
}
