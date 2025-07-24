---
title: "Accurately Perform Avoidance Actions"
description: "Accurately Perform Avoidance Actions"
weight: 21
---

## Accurately Perform Avoidance Actions
Through the following two points, the excessive operation of low-quality pod can be avoided, and the gap between the metrics and the specified watermark can be reduced faster, so as to ensure that the high-priority service is not affected
1. Sort pod

Crane implements some general sorting methods (which will be improved later):

ClassAndPriority: compare the QOSClass and class value of two pods, compare QOSClass first, and then class value; Those with high priority are ranked later and have higher priority

runningTime: compare the running time of two pods. The one with long running time is ranked later and has higher priority

If you only need to use these two sorting strategies, you can use the default sorting method: you will first compare the priority of the pod, then compare the consumption of the corresponding indicators of the pod, and then compare the running time of the pod. There is a dimension that can compare the results, that is, the sorting results of the pod

Taking the ranking of CPU usage metric as an example, it also extends some ranking strategies related to its own metric, such as the ranking of CPU usage, which will compare the priority of two pods in turn. If the priority is the same, then compare the CPU consumption. If the CPU consumption is also the same, continue to compare the extended CPU resource consumption, and finally compare the running time of pod, when there is a difference in an indicator, the comparison result can be returned: `orderedby (classandpriority, CpuUsage, extcpuusage, runningtime) Sort(pods)`

2. Refer to the watermark and pod usage to perform avoidance action
```go
//Divide all the metrics that trigger the watermark threshold into two parts according to their quantified attribute
metricsQuantified, MetricsNotQuantified := ThrottleDownWaterLine.DivideMetricsByQuantified()
// If there is a metric that cannot be quantified, obtain the metric of a throttleable with the highest actionpriority to operate on all selected pods
if len(MetricsNotThrottleQuantified) != 0 {
    highestPrioriyMetric := GetHighestPriorityThrottleAbleMetric()
    t.throttlePods(ctx, &totalReleased, highestPrioriyMetric)
} else {
    //Get the latest usage, get the gap to watermark
    ThrottoleDownGapToWaterLines = buildGapToWaterLine(ctx.getStateFunc())
    //If the real-time consumption of metric in the trigger watermark threshold cannot be obtained, chose the metric which is throttleable with the highest actionpriority to suppress all selected pods
    if ThrottoleDownGapToWaterLines.HasUsageMissedMetric() {
        highestPrioriyMetric := ThrottleDownWaterLine.GetHighestPriorityThrottleAbleMetric()
        errPodKeys = throttlePods(ctx, &totalReleased, highestPrioriyMetric)
    } else {
        var released ReleaseResource
        //Traverse the quantifiable metrics in the metrics that trigger the watermark: if the metric has a sorting method, use its sortfunc to sort the pod directly, 
        //otherwise use generalsorter to sort; Then use its corresponding operation method to operate the pod, and calculate the amount of resources released from the corresponding metric until the gap between the corresponding metric and the watermark no longer exists
        for _, m := range metricsQuantified {
            if m.SortAble {
                m.SortFunc(ThrottleDownPods)
            } else {
                GeneralSorter(ThrottleDownPods)
            }
    
            for !ThrottoleDownGapToWaterLines.TargetGapsRemoved(m) {
                for index, _ := range ThrottleDownPods {
                    released = m.ThrottleFunc(ctx, index, ThrottleDownPods, &totalReleased)
                    ThrottoleDownGapToWaterLines[m] -= released[m]
                }
            }
        }
    }
}
```
About extending user-defined metrics and sorting, it is introduced in "User-defined metrics interference detection avoidance and user-defined sorting".