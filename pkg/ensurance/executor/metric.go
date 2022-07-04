package executor

import (
	"sync"

	podinfo "github.com/gocrane/crane/pkg/ensurance/executor/pod-info"
)

type metric struct {
	// Should be consistent with metrics in collector/types/types.go
	Name WaterLineMetric

	// ActionPriority describe the priority of the metric, used to choose the highest priority metric which can be throttlable or evictable
	// when there is MetricsNotThrottleQuantified in executor process;
	// The range is 0 to 10, 10 is the highest, 0 is the lowest;
	// Some incompressible metric such as memory usage can be given a higher priority
	ActionPriority int

	SortAble bool
	SortFunc func(pods []podinfo.PodContext)

	ThrottleAble       bool
	ThrottleQuantified bool
	ThrottleFunc       func(ctx *ExecuteContext, index int, ThrottleDownPods ThrottlePods, totalReleasedResource *ReleaseResource) (errPodKeys []string, released ReleaseResource)
	RestoreFunc        func(ctx *ExecuteContext, index int, ThrottleUpPods ThrottlePods, totalReleasedResource *ReleaseResource) (errPodKeys []string, released ReleaseResource)

	EvictAble       bool
	EvictQuantified bool
	// If use goroutine to evcit, make sure to calculate release resources outside the goroutine
	EvictFunc func(wg *sync.WaitGroup, ctx *ExecuteContext, index int, totalReleasedResource *ReleaseResource, EvictPods EvictPods) (errPodKeys []string, released ReleaseResource)
}

var metricMap = make(map[WaterLineMetric]metric)

func registerMetricMap(m metric) {
	metricMap[m.Name] = m
}

func GetThrottleAbleMetricName() (throttleAbleMetricList []WaterLineMetric) {
	for _, m := range metricMap {
		if m.ThrottleAble {
			throttleAbleMetricList = append(throttleAbleMetricList, m.Name)
		}
	}
	return
}

func GetEvictAbleMetricName() (evictAbleMetricList []WaterLineMetric) {
	for _, m := range metricMap {
		if m.EvictAble {
			evictAbleMetricList = append(evictAbleMetricList, m.Name)
		}
	}
	return
}
