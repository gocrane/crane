---
title: "闲置节点推荐"
description: "闲置节点推荐功能介绍"
weight: 15
---

闲置节点推荐通过扫描节点的状态和利用率，帮助用户找到闲置的 Kubernetes node。

## 动机

在使用 Kubernetes 的过程中，常常由于污点配置、label selector、低装箱率、低利用率等因素导致部分节点出现闲置状态，浪费了大量成本，闲置节点推荐尝试帮助用户找到这部分节点来实现成本优化。

## 推荐示例

```yaml
kind: Recommendation
apiVersion: analysis.crane.io/v1alpha1
metadata:
  name: idlenodes-rule-idlenode-5jxn9
  namespace: crane-system
  labels:
    analysis.crane.io/recommendation-rule-name: idlenodes-rule
    analysis.crane.io/recommendation-rule-recommender: IdleNode
    analysis.crane.io/recommendation-rule-uid: 8921a198-7082-11ed-8b7b-246e960a8d8c
    analysis.crane.io/recommendation-target-kind: Node
    analysis.crane.io/recommendation-target-name: worker-node-1
    analysis.crane.io/recommendation-target-version: v1
    beta.kubernetes.io/arch: amd64
    beta.kubernetes.io/instance-type: bareMetal
    beta.kubernetes.io/os: linux
  ownerReferences:
    - apiVersion: analysis.crane.io/v1alpha1
      kind: RecommendationRule
      name: idlenodes-rule
      uid: 8921a198-7082-11ed-8b7b-246e960a8d8c
      controller: false
      blockOwnerDeletion: false
spec:
  targetRef:
    kind: Node
    name: worker-node-1
    apiVersion: v1
  type: IdleNode
  completionStrategy: {}
status:
  targetRef: {}
  action: Delete
  lastUpdateTime: '2022-11-30T07:46:57Z'
```

在该示例中：

- 推荐的 TargetRef 指向了 Node：worker-node-1
- 推荐类型为闲置节点推荐
- action 是 Delete，但是下线节点是复杂操作，这里只是给出建议

## 实现原理

闲置节点推荐按以下步骤完成一次推荐过程：

1. 扫描集群中所有节点和节点上的 Pod
2. 如果节点上所有 Pod 都属于 DaemonSet，则判定为闲置节点

