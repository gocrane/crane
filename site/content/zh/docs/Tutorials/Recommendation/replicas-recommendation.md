---
title: "副本数推荐"
description: "副本数推荐介绍"
weight: 13
---

Kubernetes 用户在创建应用资源时常常是基于经验值来设置副本数。通过副本数推荐的算法分析应用的真实用量推荐更合适的副本配置，您可以参考并采纳它提升集群的资源利用率。

## 实现原理

基于 Workload 历史 CPU 负载，找到过去七天内每小时负载最低的 CPU 用量，计算按50%（可配置）利用率和 Workload CPU Request 应配置的副本数

### Filter 阶段

1. 低副本数的工作负载: 过低的副本数可能推荐需求不高，关联配置: `workload-min-replicas`
2. 存在一定比例非 Running Pod 的工作负载: 如果工作负载的 Pod 大多不能正常运行，可能不适合弹性，关联配置: `pod-min-ready-seconds` | `pod-available-ratio`

### Prepare 阶段

查询过去一周的 CPU 使用量

### Recommend 阶段

1. 计算过去7天 workload 每小时使用量中位数的最低值(防止极小值影响): workload_cpu_usage_medium_min
2. 目标利用率对应的副本数:

```go
   	replicas := int32(math.Ceil(workloadCpu / (rr.TargetUtilization * float64(requestTotal) / 1000.)))
```

3. 为了防止 replicas 过小，replicas 需要大于等于 default-min-replicas

### Observe 阶段

将推荐 replicas 记录到 Metric：crane_analytics_replicas_recommendation

## 支持的资源类型

默认支持 StatefulSet 和 Deployment，但是支持所有实现了 Scale SubResource 的 Workload。

## 参数配置

| 配置项 | 默认值 | 描述              |
| ------------- |-----|-----------------|
| workload-min-replicas| 1   | 小于该值的工作负载不做弹性推荐 |
| pod-min-ready-seconds| 30  | 定义了 Pod 是否 Ready 的秒数 |
| pod-available-ratio| 0.5 | Ready Pod 比例小于该值的工作负载不做弹性推荐 |
| default-min-replicas| 1   | 最小 minReplicas  |
| cpu-target-utilization| 0.5 | 按该值计算最小副本数      |
