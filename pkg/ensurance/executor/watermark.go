package executor

import (
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
	"math"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/collector/types"
)

// WatermarkMetric defines metrics that can be measured for watermark
// Should be consistent with metrics in collector/types/types.go
type WatermarkMetric string

// Be consistent with metrics in collector/types/types.go
const (
	CpuUsage = WatermarkMetric(types.MetricNameCpuTotalUsage)
	MemUsage = WatermarkMetric(types.MetricNameMemoryTotalUsage)
)

const (
	// We can't get current use, so can't do actions precisely, just evict every evictedPod
	maxFloat float64 = math.MaxFloat64
)

// An Watermark is a min-heap of Quantity. The values come from each objectiveEnsurance.metricRule.value
type Watermark []resource.Quantity

func (w Watermark) Len() int {
	return len(w)
}

func (w Watermark) Swap(i, j int) {
	w[i], w[j] = w[j], w[i]
}

func (w *Watermark) Push(x interface{}) {
	*w = append(*w, x.(resource.Quantity))
}

func (w *Watermark) Pop() interface{} {
	old := *w
	n := len(old)
	x := old[n-1]
	*w = old[0 : n-1]
	return x
}

func (w *Watermark) PopSmallest() *resource.Quantity {
	wl := *w
	return &wl[0]
}

func (w Watermark) Less(i, j int) bool {
	cmp := w[i].Cmp(w[j])
	if cmp == -1 {
		return true
	}
	return false
}

func (w Watermark) String() string {
	str := ""
	for i := 0; i < w.Len(); i++ {
		str += w[i].String()
		str += " "
	}
	return str
}

// Watermarks 's key is the metric name, value is watermark which get from each objectiveEnsurance.metricRule.value
type Watermarks map[WatermarkMetric]*Watermark

// DivideMetricsByThrottleQuantified divide metrics by whether metrics can be throttleQuantified
func (e Watermarks) DivideMetricsByThrottleQuantified() (MetricsThrottleQuantified []WatermarkMetric, MetricsNotThrottleQuantified []WatermarkMetric) {
	for m := range e {
		if metricMap[m].ThrottleQuantified == true {
			MetricsThrottleQuantified = append(MetricsThrottleQuantified, m)
		} else {
			MetricsNotThrottleQuantified = append(MetricsNotThrottleQuantified, m)
		}
	}
	return
}

// DivideMetricsByEvictQuantified divide metrics in watermarks into can be EvictQuantified and can not be EvictQuantified
func (e Watermarks) DivideMetricsByEvictQuantified() (quantified []WatermarkMetric, notQuantified []WatermarkMetric) {
	for m := range e {
		if metricMap[m].EvictQuantified == true {
			quantified = append(quantified, m)
		} else {
			notQuantified = append(notQuantified, m)
		}
	}
	return
}

// GetHighestPriorityThrottleAbleMetric get the highest priority in metrics from watermarks
func (e Watermarks) GetHighestPriorityThrottleAbleMetric() (highestPrioriyMetric WatermarkMetric) {
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

// GetHighestPriorityEvictableMetric get the highest priority in metrics that can be Evictable
func (e Watermarks) GetHighestPriorityEvictableMetric() (highestPrioriyMetric WatermarkMetric) {
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

// Gaps key is metric name, value is the difference between usage and the smallest watermark
type Gaps map[WatermarkMetric]float64

// Only calculate gap for metrics that can be quantified
func calculateGaps(stateMap map[string][]common.TimeSeries,
	throttleExecutor *ThrottleExecutor, evictExecutor *EvictExecutor, executeExcessPercent float64) Gaps {
	result := map[WatermarkMetric]float64{}
	//throttleDownGapToWatermarks, throttleUpGapToWatermarks, eviceGapToWatermarks = make(map[WatermarkMetric]float64), make(map[WatermarkMetric]float64), make(map[WatermarkMetric]float64)

	if evictExecutor != nil {
		// Traverse EvictableMetric but not evictExecutor.EvictWatermark can make it easier when users use the wrong metric name in NodeQOS, cause this limit metrics
		// must come from EvictableMetrics
		for _, m := range metricMap {
			if !m.Evictable {
				continue
			}
			// Get the series for each metric
			series, ok := stateMap[string(m.Name)]
			if !ok {
				klog.Warningf("BuildEvictWatermarkGap: Evict Metric %s not found from collector stateMap", string(m.Name))
				// Can't get current usage, so can not do actions precisely, just evict every evictedPod;
				result[m.Name] = maxFloat
				continue
			}

			// Find the biggest used value
			var maxUsed float64
			if series[0].Samples[0].Value > maxUsed {
				maxUsed = series[0].Samples[0].Value
			}

			// Get the watermark for each metric cannot be quantified
			evictWatermark, evictExist := evictExecutor.EvictWatermark[m.Name]
			// If metric not exist in EvictWatermark, gap can't be calculated
			if !evictExist {
				delete(result, m.Name)
			} else {
				klog.V(6).Infof("BuildEvictWatermarkGap: For metrics %+v, maxUsed is %f, watermark is %f", m, maxUsed, float64(evictWatermark.PopSmallest().Value()))
				result[m.Name] = (1 + executeExcessPercent) * (maxUsed - float64(evictWatermark.PopSmallest().Value()))
			}
		}
	} else if throttleExecutor != nil {
		// Traverse ThrottleAbleMetricName but not throttleExecutor.ThrottleDownWatermark can make it easier when users use the wrong metric name in NEP, cause this limit metrics
		// must come from ThrottleAbleMetrics
		for _, m := range metricMap {
			if !m.Throttleable {
				continue
			}
			// Get the series for each metric
			series, ok := stateMap[string(m.Name)]
			if !ok {
				klog.Warningf("BuildThrottleWatermarkGap: Metric %s not found from collector stateMap", string(m.Name))
				// Can't get current usage, so can not do actions precisely, just evict every evictedPod;
				result[m.Name] = maxFloat
				continue
			}

			// Find the biggest used value
			var maxUsed float64
			if series[0].Samples[0].Value > maxUsed {
				maxUsed = series[0].Samples[0].Value
			}

			// Get the watermark for each metric in WatermarkMetricsCanBeQuantified
			throttleDownWatermark, throttleDownExist := throttleExecutor.ThrottleDownWatermark[m.Name]
			throttleUpWatermark, throttleUpExist := throttleExecutor.ThrottleUpWatermark[m.Name]

			// If a metric does not exist in ThrottleDownWatermark, throttleDownGapToWatermarks of this metric will can't be calculated
			if !throttleDownExist {
				delete(result, m.Name)
			} else {
				klog.V(6).Infof("BuildThrottleDownWatermarkGap: For metrics %s, maxUsed is %f, watermark is %f", m.Name, maxUsed, float64(throttleDownWatermark.PopSmallest().Value()))
				result[m.Name] = (1 + executeExcessPercent) * (maxUsed - float64(throttleDownWatermark.PopSmallest().Value()))
			}

			// If metric not exist in ThrottleUpWatermark, throttleUpGapToWatermarks of metric will can't be calculated
			if !throttleUpExist {
				delete(result, m.Name)
			} else {
				klog.V(6).Infof("BuildThrottleUpWatermarkGap: For metrics %s, maxUsed is %f, watermark is %f", m.Name, maxUsed, float64(throttleUpWatermark.PopSmallest().Value()))
				// Attention: different with throttleDown and evict, use watermark - used
				result[m.Name] = (1 + executeExcessPercent) * (float64(throttleUpWatermark.PopSmallest().Value()) - maxUsed)
			}
		}
	}
	return result
}

// Whether no gaps in Gaps
func (g Gaps) GapsAllRemoved() bool {
	for _, v := range g {
		if v > 0 {
			return false
		}
	}
	return true
}

// For a specified metric in Gaps, whether there still has gap
func (g Gaps) TargetGapsRemoved(metric WatermarkMetric) bool {
	val, ok := g[metric]
	if !ok || val <= 0 {
		return true
	}
	return false
}

// Whether there is a metric that can't get usage in Gaps
func (g Gaps) HasUsageMissedMetric() bool {
	for m, v := range g {
		if v == maxFloat {
			klog.V(6).Infof("Metric %s usage missed", m)
			return true
		}
	}
	return false
}
