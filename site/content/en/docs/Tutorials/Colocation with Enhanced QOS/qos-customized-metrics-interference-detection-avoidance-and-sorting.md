---
title: "Define your watermark"
description: "How to customized your watermark"
weight: 22
---

## User-defined metrics interference detection avoidance and user-defined sorting
The use of user-defined metrics interference detection avoidance and user-defined sorting is the same as the process described in the "Accurately Perform Avoidance Actions". Here is how to customize your own metrics to participate in the interference detection avoidance process

In order to better sort and accurately control metrics configured based on NodeQOS, the concept of attributes is introduced into metrics.

The attributes of metric include the following, and these fields can be realized by customized indicators:

1. Name Indicates the name of metric, which should be consistent with the metric name collected in the collector module
2. ActionPriority Indicates the priority of the metric. 0 is the lowest and 10 is the highest
3. SortAble Indicates whether the metric can be sorted. If it is true, the corresponding SortFunc needs to be implemented
4. SortFunc The corresponding sorting method. The sorting method can be arranged and combined with some general methods, and then combined with the sorting of the metric itself, which will be introduced in detail below
5. ThrottleAble Indicates whether pod can be suppressed for this metric. For example, for the metric of CPU usage, there are corresponding suppression methods, but for the metric of memory usage, pod can only be evicted, and effective suppression cannot be carried out
6. ThrottleQuantified Indicates whether the amount of resources corresponding to metric released after suppressing (restoring) a pod can be accurately calculated. We call the metric that can be accurately quantified as quantifiable, otherwise it is not quantifiable;
   For example, the CPU usage can be suppressed by limiting the CGroup usage, and the CPU usage released after suppression can be calculated by the current running value and the value after suppression; Memory usage does not belong to suppression quantifiable metric, because memory has no corresponding throttle implementation, so it is impossible to accurately measure the specific amount of memory resources released after suppressing a pod;
7. ThrottleFunc The specific method of executing throttle action. If throttle is not available, the returned released is null
8. RestoreFunc After being throttled, the specific method of performing the recovery action. If restore is not allowed, the returned released is null
9. Evictable, EvictQuantified and EvictFunc The relevant definitions of evict action are similar to those of throttle action

```go
type metric struct {
	Name WaterLineMetric

	ActionPriority int

	SortAble bool
	SortFunc func(pods []podinfo.PodContext)

	ThrottleAble      bool
	ThrottleQuantified bool
	ThrottleFunc      func(ctx *ExecuteContext, index int, ThrottleDownPods ThrottlePods, totalReleasedResource *ReleaseResource) (errPodKeys []string, released ReleaseResource)
	RestoreFunc       func(ctx *ExecuteContext, index int, ThrottleUpPods ThrottlePods, totalReleasedResource *ReleaseResource) (errPodKeys []string, released ReleaseResource)

	EvictAble      bool
	EvictQuantified bool
	EvictFunc      func(wg *sync.WaitGroup, ctx *ExecuteContext, index int, totalReleasedResource *ReleaseResource, EvictPods EvictPods) (errPodKeys []string, released ReleaseResource)
}
```

After the construction is completed, register the metric through registerMetricMap()

For the metrics that need to be customized, you can easily realize the flexible customized sorting of pod by combining the following methods with general sorting methods to represent the customized metric indicators, <metric-sort-func> represents the customized sorting strategy

```yaml
func <metric>Sorter(pods []podinfo.PodContext) {
  orderedBy(classAndPriority, <metric-sort-func>, runningTime).Sort(pods)
}
```
Among them, the following sorting method `<metric-sort-func>` needs to be implemented
`func (p1, p2 podinfo.PodContext) int32` 