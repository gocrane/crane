---
title: "资源推荐"
description: "资源推荐功能介绍"
weight: 14
---

Kubernetes 用户在创建应用资源时常常是基于经验值来设置 request 和 limit。通过资源推荐的算法分析应用的真实用量推荐更合适的资源配置，您可以参考并采纳它提升集群的资源利用率。

## 实现原理

算法模型采用了 VPA 的滑动窗口（Moving Window）算法进行推荐

1. 通过监控数据，获取 Workload 过去一周（可配置）的 CPU 和 Memory 历史用量。
2. 算法考虑数据的时效性，较新的数据采样点会拥有更高的权重。
3. CPU 推荐值基于用户设置的目标百分位值计算，Memory 推荐值基于历史数据的最大值

### Filter 阶段

没有 Pod 的工作负载: 如果工作负载没有 Pod，无法进行算法分析

### Recommend 推荐

采用 VPA 的滑动窗口（Moving Window）算法分别计算每个容器的 CPU 和 Memory 并给出对应的推荐值

### Observe 推荐

将推荐资源配置记录到 Metric：crane_analytics_replicas_recommendation

## 支持的资源类型

默认支持 StatefulSet 和 Deployment，但是支持所有实现了 Scale SubResource 的 Workload。

## 参数配置

| 配置项                         | 默认值  | 描述                             |
|-----------------------------|------|--------------------------------|
| cpu-sample-interval         | 1m   | 请求 CPU 监控数据的 Metric 采样点时间间隔    |
| cpu-request-percentile      | 0.99 | CPU 百分位值                       |
| cpu-request-margin-fraction | 0.15 | CPU 推荐值扩大系数，0.15指推荐值乘以 1.15    |
| cpu-target-utilization      | 1    | CPU 目标利用率，0.8 指推荐值除以 0.8       |
| cpu-model-history-length    | 168h | CPU 历史监控数据的时间                  |
| mem-sample-interval         | 1m   | 请求 Memory 监控数据的 Metric 采样点时间间隔 |
| mem-request-percentile      | 0.99 | Memory 百分位值                    |
| mem-request-margin-fraction | 0.15 | Memory 推荐值扩大系数，0.15指推荐值乘以 1.15 |
| mem-target-utilization      | 1    | Memory 目标利用率，0.8 指推荐值除以 0.8       |
| mem-model-history-length    | 168h | Memory 历史监控数据的时间                  |
