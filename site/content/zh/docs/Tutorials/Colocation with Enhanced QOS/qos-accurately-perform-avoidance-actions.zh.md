---
title: "精确执行回避动作"
description: "精确执行回避动作"
weight: 21
---

## 精确执行回避动作
通过如下两点进行，避免了对于低优pod的过度操作的同时能够更快地降低指标到指定水位线的差距，保障高优业务不受影响
1. 排序pod


crane实现了一些通用的排序方法（之后会更多地完善）：

classAndPriority： 比较两个pod的QOSClass和class value，优先比较QOSClass，再比较class value；priority高的排在后面优先级更高

runningTime：比较两个pod的运行时间，运行时间长的排在后面优先级更高

如果仅需使用这两个排序策略，使用默认的排序方法即可：会首先比较pod的优先级，之后比较pod对应指标的用量，之后比较pod的运行时长，有一个维度可以比较出结果即为pod的排序结果

以cpu usage指标的排序为例，还扩展了一些与自身指标相关的排序策略， 如cpu usage 使用量的排序，会依次比较两个pod的优先级，如果优先级相同的情况下，再比较cpu用量，如果cpu用量也相同的情况下继续比较扩展cpu资源用量, 最后比较pod的运行时长，当某一个指标存在差异时即可返回比较结果：`orderedBy(classAndPriority, cpuUsage, extCpuUsage, runningTime).Sort(pods)`


2. 参考水位线和pod用量执行回避动作
   ```go
   //将所有触发水位线的metrics根据其Quantified属性区分为两部分
   metricsQuantified, MetricsNotQuantified := ThrottleDownWaterLine.DivideMetricsByQuantified()
   // 如果存在不可Quantified的metric，获取具有最高ActionPriority的一个throttleAble的metric对所选择的所有pod进行操作
   if len(MetricsNotThrottleQuantified) != 0 {
       highestPrioriyMetric := GetHighestPriorityThrottleAbleMetric()
       t.throttlePods(ctx, &totalReleased, highestPrioriyMetric)
   } else {
       //获取节点和workload的最新用量，构造和水位线差距
       ThrottoleDownGapToWaterLines = buildGapToWaterLine(ctx.getStateFunc())
       //如果触发水位线中存在metric的实时用量无法获取，则获取具有最高ActionPriority的一个throttleAble的metric对所选择的所有pod进行压制操作
       if ThrottoleDownGapToWaterLines.HasUsageMissedMetric() {
           highestPrioriyMetric := ThrottleDownWaterLine.GetHighestPriorityThrottleAbleMetric()
           errPodKeys = throttlePods(ctx, &totalReleased, highestPrioriyMetric)
       } else {
           var released ReleaseResource
           //遍历触发水位线的metric中可以量化的metric：如果metric具有排序方法则直接使用其SortFunc对pod进行排序，否则使用GeneralSorter排序；
           //之后使用其对应的操作方法对pod执行操作，并计算释放出来的对应metric的资源量，直到对应metric到水位线的差距已不存在
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
关于扩展自定义指标和排序参考 "自定义指标干扰检测回避和自定义排序" 部分