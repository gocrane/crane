---
title: "动态调度器：一个基于负载感知的调度插件"
description: "动态调度器插件功能介绍"
weight: 12
---

kubernetes 的原生调度器只能通过资源请求来调度 pod，这很容易造成一系列负载不均的问题：

- 对于某些节点，实际负载与资源请求相差不大，这会导致很大概率出现稳定性问题。
- 对于其他节点来说，实际负载远小于资源请求，这将导致资源的巨大浪费。

为了解决这些问题，动态调度器根据实际的节点利用率构建了一个简单但高效的模型，并过滤掉那些负载高的节点来平衡集群。

## 设计细节

### 架构
![](/images/dynamic-scheduler-plugin.png)


如上图，动态调度器依赖于`Prometheus`和`Node-exporter`收集和汇总指标数据，它由两个组件组成：

!!! note "Note"
`Node-annotator` 目前是 `Crane-scheduler-controller`的一个模块.

- `Node-annotator`定期从 Prometheus 拉取数据，并以注释的形式在节点上用时间戳标记它们。
- `Dynamic plugin`直接从节点的注释中读取负载数据，过滤并基于简单的算法对候选节点进行评分。

###  调度策略
动态调度器提供了一个默认值[调度策略](../deploy/manifests/policy.yaml)并支持用户自定义策略。默认策略依赖于以下指标：

- `cpu_usage_avg_5m`
- `cpu_usage_max_avg_1h`
- `cpu_usage_max_avg_1d`
- `mem_usage_avg_5m`
- `mem_usage_max_avg_1h`
- `mem_usage_max_avg_1d`

在调度的`Filter`阶段，如果该节点的实际使用率大于上述任一指标的阈值，则该节点将被过滤。而在`Score`阶段，最终得分是这些指标值的加权和。

### Hot Value

在生产集群中，可能会频繁出现调度热点，因为创建 Pod 后节点的负载不能立即增加。因此，我们定义了一个额外的指标，名为`Hot Value`，表示节点最近几次的调度频率。并且节点的最终优先级是最终得分减去`Hot Value`。
