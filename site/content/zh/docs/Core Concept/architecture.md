---
title: "系统架构"
description: "整体的系统架构"
weight: 18

---

Crane 的整体架构如下：

![Crane Arch](/images/crane-arch.png)

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
- [api](https://github.com/gocrane/api) - This repository defines component-level APIs for the Crane platform.
- [fadvisor](https://github.com/gocrane/fadvisor) - Financial advisor which collect resource prices from cloud API.
- [crane-scheduler](https://github.com/gocrane/crane-scheduler) - A Kubernetes scheduler which can schedule pod based on actual node load.
- [kubectl-crane](https://github.com/gocrane/kubectl-crane) - Kubectl plugin for crane, including recommendation and cost estimate.
