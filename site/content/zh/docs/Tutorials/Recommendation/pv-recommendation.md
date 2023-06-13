---
title: "PV 推荐"
description: "PV 推荐功能介绍"
weight: 15
---

PV 推荐通过扫描集群中 PV 的运行状况，帮助用户找到闲置的 Kubernetes PV。

## 动机

通常在 Kubernetes 中我们会使用 PV + Workload 来自动创建和管理存储卷并将存储卷挂载到应用上，在日常的运营中难免会出现空闲或者空跑的存储卷，浪费了大量成本， PV 推荐尝试帮助用户找到这部分 PV 来实现成本优化。

## 推荐示例

```yaml

```

在该示例中：

- 推荐的 TargetRef 指向了 PV:
- 推荐类型为 PV 推荐
- action 是 Delete，这里只是给出建议

## 实现原理

PV 推荐按以下步骤完成一次推荐过程：

1. 扫描集群中所有 PV，找到 PV 对应的 Pod 列表
2. 如果 PV 没有对应的 PVC，则判断为闲置 PV
3. 如果没有 Pod 关联这个 PV 和 PVC，则判断为闲置 PVC

## 参数配置

目前 PV 推荐没有参数配置。

如何更新推荐的配置请参考：[**推荐框架**](/zh-cn/docs/tutorials/recommendation/recommendation-framework)