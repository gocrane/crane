package executor

import (
	"math"
	"reflect"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/collector/types"
)

// Metrics that can be measured for waterLine
// Should be consistent with metrics in collector/types/types.go
type WaterLineMetric string

// Be consistent with metrics in collector/types/types.go
const (
	CpuUsage = WaterLineMetric(types.MetricNameCpuTotalUsage)
	MemUsage = WaterLineMetric(types.MetricNameMemoryTotalUsage)
)

const (
	// We can't get current use, so can't do actions precisely, just evict every evictedPod
	MissedCurrentUsage float64 = math.MaxFloat64
)

// An WaterLine is a min-heap of Quantity. The values come from each objectiveEnsurance.metricRule.value
type WaterLine []resource.Quantity

func (w WaterLine) Len() int {
	return len(w)
}

func (w WaterLine) Swap(i, j int) {
	w[i], w[j] = w[j], w[i]
}

func (w *WaterLine) Push(x interface{}) {
	*w = append(*w, x.(resource.Quantity))
}

func (w *WaterLine) Pop() interface{} {
	old := *w
	n := len(old)
	x := old[n-1]
	*w = old[0 : n-1]
	return x
}

func (w *WaterLine) PopSmallest() *resource.Quantity {
	wl := *w
	return &wl[0]
}

func (w WaterLine) Less(i, j int) bool {
	cmp := w[i].Cmp(w[j])
	if cmp == -1 {
		return true
	}
	return false
}

func (w WaterLine) String() string {
	str := ""
	for i := 0; i < w.Len(); i++ {
		str += w[i].String()
		str += " "
	}
	return str
}

// WaterLines 's key is the metric name, value is waterline which get from each objectiveEnsurance.metricRule.value
type WaterLines map[WaterLineMetric]*WaterLine

// DivideMetricsByThrottleQuantified divide metrics by whether metrics can be throttleQuantified
func (e WaterLines) DivideMetricsByThrottleQuantified() (MetricsThrottleQuantified []WaterLineMetric, MetricsNotThrottleQuantified []WaterLineMetric) {
	for m := range e {
		if metricMap[m].ThrottleQuantified == true {
			MetricsThrottleQuantified = append(MetricsThrottleQuantified, m)
		} else {
			MetricsNotThrottleQuantified = append(MetricsNotThrottleQuantified, m)
		}
	}
	return
}

// DivideMetricsByEvictQuantified divide metrics in waterlines into can be EvictQuantified and can not be EvictQuantified
func (e WaterLines) DivideMetricsByEvictQuantified() (metricsEvictQuantified []WaterLineMetric, metricsNotEvictQuantified []WaterLineMetric) {
	for m := range e {
		if metricMap[m].EvictQuantified == true {
			metricsEvictQuantified = append(metricsEvictQuantified, m)
		} else {
			metricsNotEvictQuantified = append(metricsNotEvictQuantified, m)
		}
	}
	return
}

// GetHighestPriorityThrottleAbleMetric get the highest priority in metrics from waterlines
func (e WaterLines) GetHighestPriorityThrottleAbleMetric() (highestPrioriyMetric WaterLineMetric) {
	highestActionPriority := 0
	for m := range e {
		if metricMap[m].Throttleable == true {
			if metricMap[m].ActionPriority >= highestActionPriority {
				highestPrioriyMetric = m
				highestActionPriority = metricMap[m].ActionPriority
			}
		}
	}
	return
}

// GetHighestPriorityEvictAbleMetric get the highest priority in metrics that can be Evictable
func (e WaterLines) GetHighestPriorityEvictAbleMetric() (highestPrioriyMetric WaterLineMetric) {
	highestActionPriority := 0
	for m := range e {
		if metricMap[m].Evictable == true {
			if metricMap[m].ActionPriority >= highestActionPriority {
				highestPrioriyMetric = m
				highestActionPriority = metricMap[m].ActionPriority
			}
		}
	}
	return
}

// GapToWaterLines's key is metric name, value is the difference between usage and the smallest waterline
type GapToWaterLines map[WaterLineMetric]float64

// Only calculate gap for metrics that can be quantified
func buildGapToWaterLine(stateMap map[string][]common.TimeSeries,
	throttleExecutor ThrottleExecutor, evictExecutor EvictExecutor, executeExcessPercent float64) (
	throttleDownGapToWaterLines, throttleUpGapToWaterLines, eviceGapToWaterLines GapToWaterLines) {

	throttleDownGapToWaterLines, throttleUpGapToWaterLines, eviceGapToWaterLines = make(map[WaterLineMetric]float64), make(map[WaterLineMetric]float64), make(map[WaterLineMetric]float64)

	if !reflect.DeepEqual(evictExecutor, EvictExecutor{}) {
		// Traverse EvictAbleMetric but not evictExecutor.EvictWaterLine can make it easier when users use the wrong metric name in NEP, cause this limit metrics
		// must come from EvictAbleMetrics
		for _, m := range GetEvictAbleMetricName() {
			// Get the series for each metric
			series, ok := stateMap[string(m)]
			if !ok {
				klog.Warningf("BuildEvictWaterLineGap: Evict Metric %s not found from collector stateMap", string(m))
				// Can't get current usage, so can not do actions precisely, just evict every evictedPod;
				eviceGapToWaterLines[m] = MissedCurrentUsage
				continue
			}

			// Find the biggest used value
			var maxUsed float64
			if series[0].Samples[0].Value > maxUsed {
				maxUsed = series[0].Samples[0].Value
			}

			// Get the waterLine for each metric in WaterLineMetricsCanBeQuantified
			evictWaterLine, evictExist := evictExecutor.EvictWaterLine[m]

			// If metric not exist in EvictWaterLine, eviceGapToWaterLines of metric will can't be calculated
			if !evictExist {
				delete(eviceGapToWaterLines, m)
			} else {
				klog.V(6).Infof("BuildEvictWaterLineGap: For metrics %s, maxUsed is %f, waterline is %f", m, maxUsed, float64(evictWaterLine.PopSmallest().Value()))
				eviceGapToWaterLines[m] = (1 + executeExcessPercent) * (maxUsed - float64(evictWaterLine.PopSmallest().Value()))
			}
		}
	}

	if !reflect.DeepEqual(throttleExecutor, ThrottleExecutor{}) {
		// Traverse ThrottleAbleMetricName but not throttleExecutor.ThrottleDownWaterLine can make it easier when users use the wrong metric name in NEP, cause this limit metrics
		// must come from ThrottleAbleMetrics
		for _, m := range GetThrottleAbleMetricName() {
			// Get the series for each metric
			series, ok := stateMap[string(m)]
			if !ok {
				klog.Warningf("BuildThrottleWaterLineGap: Metric %s not found from collector stateMap", string(m))
				// Can't get current usage, so can not do actions precisely, just evict every evictedPod;
				throttleDownGapToWaterLines[m] = MissedCurrentUsage
				throttleUpGapToWaterLines[m] = MissedCurrentUsage
				continue
			}

			// Find the biggest used value
			var maxUsed float64
			if series[0].Samples[0].Value > maxUsed {
				maxUsed = series[0].Samples[0].Value
			}

			// Get the waterLine for each metric in WaterLineMetricsCanBeQuantified
			throttleDownWaterLine, throttleDownExist := throttleExecutor.ThrottleDownWaterLine[m]
			throttleUpWaterLine, throttleUpExist := throttleExecutor.ThrottleUpWaterLine[m]

			// If a metric does not exist in ThrottleDownWaterLine, throttleDownGapToWaterLines of this metric will can't be calculated
			if !throttleDownExist {
				delete(throttleDownGapToWaterLines, m)
			} else {
				klog.V(6).Infof("BuildThrottleDownWaterLineGap: For metrics %s, maxUsed is %f, waterline is %f", m, maxUsed, float64(throttleDownWaterLine.PopSmallest().Value()))
				throttleDownGapToWaterLines[m] = (1 + executeExcessPercent) * (maxUsed - float64(throttleDownWaterLine.PopSmallest().Value()))
			}

			// If metric not exist in ThrottleUpWaterLine, throttleUpGapToWaterLines of metric will can't be calculated
			if !throttleUpExist {
				delete(throttleUpGapToWaterLines, m)
			} else {
				klog.V(6).Infof("BuildThrottleUpWaterLineGap: For metrics %s, maxUsed is %f, waterline is %f", m, maxUsed, float64(throttleUpWaterLine.PopSmallest().Value()))
				// Attention: different with throttleDown and evict, use waterline - used
				throttleUpGapToWaterLines[m] = (1 + executeExcessPercent) * (float64(throttleUpWaterLine.PopSmallest().Value()) - maxUsed)
			}
		}
	}
	return
}

// Whether no gaps in GapToWaterLines
func (g GapToWaterLines) GapsAllRemoved() bool {
	for _, v := range g {
		if v > 0 {
			return false
		}
	}
	return true
}

// For a specified metric in GapToWaterLines, whether there still has gap
func (g GapToWaterLines) TargetGapsRemoved(metric WaterLineMetric) bool {
	val, ok := g[metric]
	if !ok || val <= 0 {
		return true
	}
	return false
}

// Whether there is a metric that can't get usage in GapToWaterLines
func (g GapToWaterLines) HasUsageMissedMetric() bool {
	for m, v := range g {
		if v == MissedCurrentUsage {
			klog.V(6).Infof("Metric %s usage missed", m)
			return true
		}
	}
	return false
}
