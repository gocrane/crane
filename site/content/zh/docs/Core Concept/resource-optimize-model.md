---
title: "应用资源优化模型"
description: "Application Resource Optimize Model"
weight: 11

---

资源优化是 FinOps 中常见的优化手段，我们基于 Kubernetes 应用的特点总结出云原生应用的**资源优化模型**：

![Resource Model](/images/resource-model.png)

图中五条线从上到下分别是：

1. 节点容量：集群中所有节点的资源总量，对应集群的 Capacity
2. 已分配：应用申请的资源总量，对应 Pod Request
3. 周峰值：应用在过去一段时间内资源用量的峰值。周峰值可以预测未来一段时间内的资源使用，通过周峰值配置资源规格的安全性较高，普适性更强
4. 日均峰值：应用在近一天内资源用量的峰值
5. 均值：应用的平均资源用量，对应 Usage

其中资源的闲置分两类：
1. Resource Slack：Capacity 和 Request 之间的差值
2. Usage Slack：Request 和 Usage 之间的差值

Total Slack = Resource Slack + Usage Slack

资源优化的目标是 **减少 Resource Slack 和 Usage Slack**。模型中针对如何一步步减少浪费提供了四个步骤，从上到下分别是：

1. 提升装箱率：提升装箱率能够让 Capacity 和 Request 更加接近。手段有很多，例如：[动态调度器](/zh-cn/docs/tutorials/scheduling-pods-based-on-actual-node-load)、腾讯云的云原生节点的节点放大功能等 
2. 业务规格调整减少资源锁定：根据周峰值资源用量调整业务规格使的 Request 可以减少到周峰值线。[资源推荐](/zh-cn/docs/tutorials/recommendation/resource-recommendation)和[副本推荐](/zh-cn/docs/tutorials/recommendation/replicas-recommendation)可以帮助应用实现此目标。
3. 业务规格调整+扩缩容兜底流量突发：在规格优化的基础上再通过 HPA 兜底突发流量使的 Request 可以减少到日均峰值线。此时 HPA 的目标利用率偏低，仅为应对突发流量，绝大多数时间内不发生自动弹性。[弹性推荐](/zh-cn/docs/tutorials/recommendation/hpa-recommendation)可以扫描出适合做弹性的应用并提供HPA配置。
4. 业务规格调整+扩缩容应对日常流量变化：在规格优化的基础上再通过 HPA 应用日常流量使的 Request 可以减少到均值。此时 HPA 的目标利用率等于应用的平均利用率。[EHPA](/zh-cn/docs/tutorials/using-effective-hpa-to-scaling-with-effectiveness)实现了基于预测的水平弹性，帮助更多应用实现智能弹性。