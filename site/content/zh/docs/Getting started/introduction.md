---
title: "介绍"
description: "Crane 介绍"
weight: 10

---

## Crane 是什么

Crane 是一个基于 FinOps 的云资源分析与成本优化平台。它的愿景是在保证客户应用运行质量的前提下实现极致的降本。

**如何在 Crane 中开启成本优化之旅？**

1. **成本展示**: Kubernetes 资源( Deployments, StatefulSets )的多维度聚合与展示。
2. **成本分析**: 周期性的分析集群资源的状态并提供优化建议。
3. **成本优化**: 通过丰富的优化工具更新配置达成降本的目标。

<iframe src="https://user-images.githubusercontent.com/35299017/186680122-d7756b47-06be-44cb-8553-1957eaa3ed45.mp4"
scrolling="no" border="0" frameborder="no" framespacing="0" allowfullscreen="true" width="1000" height="600"></iframe>

Crane Dashboard **在线 Demo**: http://dashboard.gocrane.io/

## Main Features

![Crane Overview"](/images/crane-overview.png)

**成本可视化和优化评估**

- 提供一组 Exporter 计算集群云资源的计费和账单数据并存储到你的监控系统，比如 Prometheus。
- 多维度的成本洞察，优化评估。通过 `Cloud Provider` 支持多云计费。

**推荐框架**

提供了一个可扩展的推荐框架以支持多种云资源的分析，内置了多种推荐器：资源推荐，副本推荐，闲置资源推荐。[了解更多](/zh-cn/docs/tutorials/recommendation)。

**基于预测的水平弹性器**

EffectiveHorizontalPodAutoscaler 支持了预测驱动的弹性。它基于社区 HPA 做底层的弹性控制，支持更丰富的弹性触发策略（预测，观测，周期），让弹性更加高效，并保障了服务的质量。[了解更多](/zh-cn/docs/tutorials/using-effective-hpa-to-scaling-with-effectiveness)。

**负载感知的调度器**

动态调度器根据实际的节点利用率构建了一个简单但高效的模型，并过滤掉那些负载高的节点来平衡集群。[了解更多](/zh-cn/docs/tutorials/scheduling-pods-based-on-actual-node-load)。

**基于 QoS 的混部**


## 架构

Crane 的整体架构如下：

![Crane Arch"](/images/crane-arch.png)

**Craned**

Craned 是 Crane 的最核心组件，它管理了 CRDs 的生命周期以及API。Craned 通过 `Deployment` 方式部署且由两个容器组成：
- Craned: 运行了 Operators 用来管理 CRDs，向 Dashboard 提供了 WebApi，Predictors 提供了 TimeSeries API
- Dashboard: 基于 TDesign's Starter 脚手架研发的前端项目，提供了易于上手的产品功能

**Fadvisor**

Fadvisor 提供一组 Exporter 计算集群云资源的计费和账单数据并存储到你的监控系统，比如 Prometheus。Fadvisor 通过 `Cloud Provider` 支持了多云计费的 API。

**Metric Adapter**

Metric Adapter 实现了一个 `Custom Metric Apiserver`. Metric Adapter 读取 CRDs 信息并提供基于 `Custom/External Metric API` 的 HPA Metric 的数据。

**Crane Agent**

Crane Agent 通过 `DaemonSet` 部署在集群的节点上。

## Repositories

Crane is composed of the following components:

- [craned](https://github.com/gocrane/crane/tree/main/cmd/craned) - main crane control plane.
- [metric-adaptor](https://github.com/gocrane/crane/tree/main/cmd/metric-adapter) - Metric server for driving the scaling.
- [crane-agent](https://github.com/gocrane/crane/tree/main/cmd/crane-agent) - Ensure critical workloads SLO based on abnormally detection.
- [gocrane/api](https://github.com/gocrane/api) - This repository defines component-level APIs for the Crane platform.
- [gocrane/fadvisor](https://github.com/gocrane/fadvisor) - Financial advisor which collect resource prices from cloud API.
- [gocrane/crane-scheduler](https://github.com/gocrane/crane-scheduler) - A Kubernetes scheduler which can schedule pod based on actual node load.
