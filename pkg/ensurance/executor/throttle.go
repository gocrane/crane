package executor

import (
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	podinfo "github.com/gocrane/crane/pkg/ensurance/executor/podinfo"
	execsort "github.com/gocrane/crane/pkg/ensurance/executor/sort"
	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/metrics"
)

const (
	MaxUpQuota          = 60 * 1000 // 60CU
	CpuQuotaCoefficient = 1000.0
	MaxRatio            = 100.0
)

type ThrottleExecutor struct {
	ThrottleDownPods ThrottlePods
	ThrottleUpPods   ThrottlePods
	// All metrics(not only metrics that can be quantified) metioned in triggerd NodeQOS and their corresponding watermarks
	ThrottleDownWatermark Watermarks
	ThrottleUpWatermark   Watermarks
}

type ThrottlePods []podinfo.PodContext

func (t ThrottlePods) Find(podTypes types.NamespacedName) int {
	for i, v := range t {
		if v.Key == podTypes {
			return i
		}
	}

	return -1
}

func Reverse(t ThrottlePods) []podinfo.PodContext {
	throttle := []podinfo.PodContext(t)
	l := len(throttle)
	for i := 0; i < l/2; i++ {
		throttle[i], throttle[l-i-1] = throttle[l-i-1], throttle[i]
	}
	return throttle
}

func (t *ThrottleExecutor) Avoid(ctx *ExecuteContext) error {
	var start = time.Now()
	metrics.UpdateLastTimeWithSubComponent(string(known.ModuleActionExecutor), string(metrics.SubComponentThrottle), metrics.StepAvoid, start)
	defer metrics.UpdateDurationFromStartWithSubComponent(string(known.ModuleActionExecutor), string(metrics.SubComponentThrottle), metrics.StepAvoid, start)

	klog.V(6).Infof("ThrottleExecutor avoid, %#v", *t)

	if len(t.ThrottleDownPods) == 0 {
		metrics.UpdateExecutorStatus(metrics.SubComponentThrottle, metrics.StepAvoid, 0)
	} else {
		metrics.UpdateExecutorStatus(metrics.SubComponentThrottle, metrics.StepAvoid, 1.0)
		metrics.ExecutorStatusCounterInc(metrics.SubComponentThrottle, metrics.StepAvoid)
	}

	var errPodKeys, errKeys []string
	// TODO: totalReleasedResource used for prom metrics
	totalReleased := ReleaseResource{}

	/* The step to throttle:
	1. If ThrottleDownWatermark has metrics that can't be quantified, select a throttleable metric which has the highest action priority, use its throttlefunc to throttle all ThrottleDownPods, then return
	2. Get the gaps between current usage and watermarks
		2.1 If there is a metric that can't get current usage, select a throttleable metric which has the highest action priority, use its throttlefunc to throttle all ThrottleDownPods, then return
		2.2 Traverse metrics that can be quantified, if there is a gap for the metric, then sort candidate pods by its SortFunc if exists, otherwise use GeneralSorter by default.
	       Then throttle sorted pods one by one util there is no gap to watermark
	*/
	metricsThrottleQuantified, MetricsNotThrottleQuantified := t.ThrottleDownWatermark.DivideMetricsByThrottleQuantified()

	// There is a metric that can't be ThrottleQuantified, so throttle all selected pods
	if len(MetricsNotThrottleQuantified) != 0 {
		klog.V(6).Info("ThrottleDown: There is a metric that can't be ThrottleQuantified")

		highestPriorityMetric := t.ThrottleDownWatermark.GetHighestPriorityThrottleAbleMetric()
		if highestPriorityMetric != "" {
			klog.V(6).Infof("The highestPriorityMetric is %s", highestPriorityMetric)
			errPodKeys = t.throttlePods(ctx, &totalReleased, highestPriorityMetric)
		}
	} else {
		ctx.ToBeThrottleDown = calculateGaps(ctx.stateMap, t, nil, ctx.executeExcessPercent)

		if ctx.ToBeThrottleDown.HasUsageMissedMetric() {
			klog.V(6).Info("There is a metric usage missed")
			// todo remove highest priority
			highestPriorityMetric := t.ThrottleDownWatermark.GetHighestPriorityThrottleAbleMetric()
			if highestPriorityMetric != "" {
				errPodKeys = t.throttlePods(ctx, &totalReleased, highestPriorityMetric)
			}
		} else {
			// The metrics in ToBeThrottleDown are all in WatermarkMetricsCanBeQuantified and has current usage, then throttle precisely
			var released ReleaseResource
			for _, m := range metricsThrottleQuantified {
				klog.V(6).Infof("ThrottleDown precisely on metric %s", m)
				if metricMap[m].Sortable {
					metricMap[m].SortFunc(t.ThrottleDownPods)
				} else {
					execsort.GeneralSorter(t.ThrottleDownPods)
				}

				klog.V(6).Info("After sort, the sequence to throttle is ")
				for _, pc := range t.ThrottleDownPods {
					klog.V(6).Info(pc.Key.String(), pc.ContainerCPUUsages)
				}

				for index := 0; !ctx.ToBeThrottleDown.TargetGapsRemoved(m) && index < len(t.ThrottleDownPods); index++ {
					klog.V(6).Infof("For metric %s, there is still gap to watermarks: %f", m, ctx.ToBeThrottleDown[m])

					errKeys, released = metricMap[m].ThrottleFunc(ctx, index, t.ThrottleDownPods, &totalReleased)
					klog.V(6).Infof("ThrottleDown pods %s, released %f resource", t.ThrottleDownPods[index].Key, released[m])
					errPodKeys = append(errPodKeys, errKeys...)

					ctx.ToBeThrottleDown[m] -= released[m]
				}
			}
		}
	}

	if len(errPodKeys) != 0 {
		return fmt.Errorf("some pod throttle failed,err: %s", strings.Join(errPodKeys, ";"))
	}

	return nil
}

func (t *ThrottleExecutor) throttlePods(ctx *ExecuteContext, totalReleasedResource *ReleaseResource, m WatermarkMetric) (errPodKeys []string) {
	for i := range t.ThrottleDownPods {
		errKeys, _ := metricMap[m].ThrottleFunc(ctx, i, t.ThrottleDownPods, totalReleasedResource)
		errPodKeys = append(errPodKeys, errKeys...)
	}
	return
}

func (t *ThrottleExecutor) Restore(ctx *ExecuteContext) error {
	var start = time.Now()
	metrics.UpdateLastTimeWithSubComponent(string(known.ModuleActionExecutor), string(metrics.SubComponentThrottle), metrics.StepRestore, start)
	defer metrics.UpdateDurationFromStartWithSubComponent(string(known.ModuleActionExecutor), string(metrics.SubComponentThrottle), metrics.StepRestore, start)

	klog.V(6).Infof("ThrottleExecutor restore, %v", *t)

	if len(t.ThrottleUpPods) == 0 {
		metrics.UpdateExecutorStatus(metrics.SubComponentThrottle, metrics.StepRestore, 0)
		return nil
	}

	metrics.UpdateExecutorStatus(metrics.SubComponentThrottle, metrics.StepRestore, 1.0)
	metrics.ExecutorStatusCounterInc(metrics.SubComponentThrottle, metrics.StepRestore)

	var errPodKeys, errKeys []string
	// TODO: totalReleasedResource used for prom metrics
	totalReleased := ReleaseResource{}

	/* The step to restore:
	1. If ThrottleUpWatermark has metrics that can't be quantified, select a throttleable metric which has the highest action priority, use its RestoreFunc to restore all ThrottleUpPods, then return
	2. Get the gaps between current usage and watermarks
		2.1 If there is a metric that can't get current usage, select a throttleable metric which has the highest action priority, use its RestoreFunc to restore all ThrottleUpPods, then return
		2.2 Traverse metrics that can be quantified, if there is a gap for the metric, then sort candidate pods by its SortFunc if exists, otherwise use GeneralSorter by default.
	       Then restore sorted pods one by one util there is no gap to watermark
	*/
	metricsThrottleQuantified, MetricsNotThrottleQuantified := t.ThrottleUpWatermark.DivideMetricsByThrottleQuantified()

	// There is a metric that can't be ThrottleQuantified, so restore all selected pods
	if len(MetricsNotThrottleQuantified) != 0 {
		klog.V(6).Info("ThrottleUp: There is a metric that can't be ThrottleQuantified")

		highestPrioriyMetric := t.ThrottleUpWatermark.GetHighestPriorityThrottleAbleMetric()
		if highestPrioriyMetric != "" {
			klog.V(6).Infof("The highestPrioriyMetric is %s", highestPrioriyMetric)
			errPodKeys = t.restorePods(ctx, &totalReleased, highestPrioriyMetric)
		}
	} else {
		ctx.ToBeThrottleUp = calculateGaps(ctx.stateMap, t, nil, ctx.executeExcessPercent)

		if ctx.ToBeThrottleUp.HasUsageMissedMetric() {
			klog.V(6).Info("There is a metric usage missed")
			highestPrioriyMetric := t.ThrottleUpWatermark.GetHighestPriorityThrottleAbleMetric()
			if highestPrioriyMetric != "" {
				errPodKeys = t.restorePods(ctx, &totalReleased, highestPrioriyMetric)
			}
		} else {
			// The metrics in ToBeThrottleUp are all in WatermarkMetricsCanBeQuantified and has current usage, then throttle precisely
			var released ReleaseResource
			for _, m := range metricsThrottleQuantified {
				klog.V(6).Infof("ThrottleUp precisely on metric %s", m)
				if metricMap[m].Sortable {
					metricMap[m].SortFunc(t.ThrottleUpPods)
				} else {
					execsort.GeneralSorter(t.ThrottleUpPods)
				}
				//t.ThrottleUpPods = Reverse(t.ThrottleUpPods)

				klog.V(6).Info("After sort, the sequence to throttle is ")
				for _, pc := range t.ThrottleUpPods {
					klog.V(6).Info(pc.Key.String())
				}

				for index := 0; !ctx.ToBeThrottleUp.TargetGapsRemoved(m) && index < len(t.ThrottleUpPods); index++ {
					klog.V(6).Infof("For metric %s, there is still gap to watermarks: %f", m, ctx.ToBeThrottleUp[m])

					errKeys, released = metricMap[m].RestoreFunc(ctx, index, t.ThrottleUpPods, &totalReleased)
					klog.V(6).Infof("ThrottleUp pods %s, released %f resource", t.ThrottleUpPods[index].Key, released[m])
					errPodKeys = append(errPodKeys, errKeys...)

					ctx.ToBeThrottleUp[m] -= released[m]
				}
			}
		}
	}

	if len(errPodKeys) != 0 {
		return fmt.Errorf("some pod throttle restore failed,err: %s", strings.Join(errPodKeys, ";"))
	}

	return nil
}

func (t *ThrottleExecutor) restorePods(ctx *ExecuteContext, totalReleasedResource *ReleaseResource, m WatermarkMetric) (errPodKeys []string) {
	for i := range t.ThrottleUpPods {
		errKeys, _ := metricMap[m].RestoreFunc(ctx, i, t.ThrottleDownPods, totalReleasedResource)
		errPodKeys = append(errPodKeys, errKeys...)
	}
	return
}
