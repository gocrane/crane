---
title: "PV Recommendation"
description: "Introduce for PV Recommendation"
weight: 17
---

PV 推荐通过扫描集群中 PV 的运行状况，帮助用户找到闲置的 Kubernetes PV。

## 动机

通常在 Kubernetes 中我们会使用 PV + Workload 来自动创建和管理存储卷并将存储卷挂载到应用上，在日常的运营中难免会出现空闲或者空跑的存储卷，浪费了大量成本， PV 推荐尝试帮助用户找到这部分 PV 来实现成本优化。

## 推荐示例

```yaml
apiVersion: analysis.crane.io/v1alpha1
kind: Recommendation
metadata:
  annotations:
    analysis.crane.io/last-start-time: "2023-06-14 08:55:25"
    analysis.crane.io/message: Success
    analysis.crane.io/run-number: "653"
  labels:
    analysis.crane.io/recommendation-rule-name: persistentvolumes-rule
    analysis.crane.io/recommendation-rule-recommender: Volume
    analysis.crane.io/recommendation-rule-uid: 39d30abe-4c7f-4e65-b961-b00ec7776b45
    analysis.crane.io/recommendation-target-kind: PersistentVolume
    analysis.crane.io/recommendation-target-name: pvc-6ce24277-24e9-4fcf-8e8a-f9bdb5694134
    analysis.crane.io/recommendation-target-namespace: ""
    analysis.crane.io/recommendation-target-version: v1
  name: persistentvolumes-rule-volume-5r9zn
  namespace: crane-system
  ownerReferences:
    - apiVersion: analysis.crane.io/v1alpha1
      blockOwnerDeletion: false
      controller: false
      kind: RecommendationRule
      name: persistentvolumes-rule
      uid: 39d30abe-4c7f-4e65-b961-b00ec7776b45
spec:
  adoptionType: StatusAndAnnotation
  completionStrategy:
    completionStrategyType: Once
  targetRef:
    apiVersion: v1
    kind: PersistentVolume
    name: pvc-6ce24277-24e9-4fcf-8e8a-f9bdb5694134
  type: Volume
status:
  action: Delete
  description: It is an Orphan Volumes
  lastUpdateTime: "2023-06-14T08:55:25Z"
```

在该示例中：

- 推荐的 TargetRef 指向了 PV: pvc-6ce24277-24e9-4fcf-8e8a-f9bdb5694134
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