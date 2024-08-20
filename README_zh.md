# Crane: Cloud Resource Analytics and Economics

[![Go Report Card](https://goreportcard.com/badge/github.com/gocrane/crane)](https://goreportcard.com/report/github.com/gocrane/crane)
[![GoDoc](https://godoc.org/github.com/gocrane/crane?status.svg)](https://godoc.org/github.com/gocrane/crane)
[![License](https://img.shields.io/github/license/gocrane/crane)](https://www.apache.org/licenses/LICENSE-2.0.html)
![GoVersion](https://img.shields.io/github/go-mod/go-version/gocrane/crane)

<div align="center">

<img alt="Crane logo" height="100" src="docs/images/crane.svg" title="Crane" width="200"/>

</div>

---

## Crane 是什么

> [English](README.md) | 中文

Crane 是一个基于 FinOps 的云资源分析与成本优化平台。它的愿景是在保证客户应用运行质量的前提下实现极致的降本。

<div align="center">

<img alt="fcs logo" height="200" src="docs/images/Crane-FinOps-Certified-Solution.png" title="FinOps Certified Solution" width="200"/>

</div>

Crane 是 [FinOps 基金会](https://www.finops.org/)认证的[云优化方案](https://www.finops.org/certifications/finops-certified-solution/)。

**如何在 Crane 中开启成本优化之旅？**

1. **成本展示**: Kubernetes 资源( Deployments, StatefulSets )的多维度聚合与展示。
2. **成本分析**: 周期性的分析集群资源的状态并提供优化建议。
3. **成本优化**: 通过丰富的优化工具更新配置达成降本的目标。

https://user-images.githubusercontent.com/35299017/186680122-d7756b47-06be-44cb-8553-1957eaa3ed45.mp4

Crane Dashboard **在线 Demo**: http://dashboard.gocrane.io/

## Main Features

<img alt="Crane Overview" height="330" src="docs/images/crane-overview.png" width="900"/>

**成本可视化和优化评估**

- 提供一组 Exporter 计算集群云资源的计费和账单数据并存储到你的监控系统，比如 Prometheus。
- 多维度的成本洞察，优化评估。通过 `Cloud Provider` 支持多云计费。

**推荐框架**

提供了一个可扩展的推荐框架以支持多种云资源的分析，内置了多种推荐器：资源推荐，副本推荐，闲置资源推荐。[了解更多](https://gocrane.io/zh-cn/docs/tutorials/recommendation/)

**基于预测的水平弹性器**

EffectiveHorizontalPodAutoscaler 支持了预测驱动的弹性。它基于社区 HPA 做底层的弹性控制，支持更丰富的弹性触发策略（预测，观测，周期），让弹性更加高效，并保障了服务的质量。[了解更多](https://gocrane.io/zh-cn/docs/tutorials/using-effective-hpa-to-scaling-with-effectiveness/)

**负载感知的调度器**

动态调度器根据实际的节点利用率构建了一个简单但高效的模型，并过滤掉那些负载高的节点来平衡集群。[了解更多](https://gocrane.io/zh-cn/docs/tutorials/scheduling-pods-based-on-actual-node-load/)

**基于 QOS 的混部**

QOS相关能力保证了运行在 Kubernetes 上的 Pod 的稳定性。具有多维指标条件下的干扰检测和主动回避能力，支持精确操作和自定义指标接入；具有预测算法增强的弹性资源超卖能力，复用和限制集群内的空闲资源；具备增强的旁路cpuset管理能力，在绑核的同时提升资源利用效率。[了解更多](docs/tutorials/using-qos-ensurance.zh.md)。

## 架构

Crane 的整体架构如下：

<img alt="Crane Overview" height="550" src="docs/images/crane-arch.png"/>

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

## 快速开始

- [介绍](https://gocrane.io/zh-cn/docs/getting-started/introduction/)
- [快速开始](https://gocrane.io/zh-cn/docs/getting-started/installation/quick-start/)
- [教程](https://gocrane.io/zh-cn/docs/tutorials/)

## 文档

完整的文档请[查看](https://gocrane.io/zh-cn)。

## 社区

- Slack(English): [https://gocrane.slack.com](https://join.slack.com/t/gocrane/shared_invite/zt-1k3beos1i-ejN6sV0jx5_MAkKRbl~MFQ)

- 微信群: 

<img alt="Wechat" src="https://user-images.githubusercontent.com/6251116/226240172-53bae906-3abc-4b04-89d5-eee11c13faaa.png" title="Wechat" width="200"/>

<img alt="Wechat" src="docs/images/wechat.jpeg" title="Wechat" width="200"/>

添加微信后回复 "Crane"，小助手会定期将您加入微信群。

- 社区双周会(APAC, Chinese)
  - [Meeting Link](https://meeting.tencent.com/dm/SjY20wCJHy5F)
  - [Meeting Notes](https://doc.weixin.qq.com/doc/w3_AHMAlwa_AFU7PT58rVhTFKXV0maR6?scode=AJEAIQdfAAo0gvbrCIAHMAlwa_AFU)
  - [Video Records](https://www.wolai.com/33xC4HB1JXCCH1x8umfioS)

## RoadMap

[了解更多](./docs/roadmaps/roadmap-2022.md)。

## 如何贡献

欢迎参与贡献 Crane 项目。请参考 [CONTRIBUTING](./CONTRIBUTING.md) 了解如何参与贡献。

关于如何参与 Crane 的开发，你可以参考 [开发文档](./docs/developer-guide.md)。

## 行为准则

Crane 采用了 [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md).
