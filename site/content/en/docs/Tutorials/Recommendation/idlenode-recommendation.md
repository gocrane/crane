---
title: "IdleNode Recommendation"
description: "Introduce for IdleNode Recommendation"
weight: 15
---

By scanning the status and utilization of nodes, the idle node recommendation helps users to find idle Kubernetes nodes.

## Motivation

In Kubernetes cluster, some nodes often idle due to such factors as node taint, label selector, low packing rate and low utilization rate, which wastes a lot of costs. IdleNode recommendation tries to help users find these nodes to reduce cost.

## Example

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

In this example：

- Recommendation's TargetRef Point to Node：worker-node-1
- Recommendation type is IdleNode 
- action is Delete，but offline a node is a complicated operation, we only give recommended advise.

## Implement

Perform the following steps to complete a recommendation process for idle nodes:

1. Scan all nodes and pods in the cluster
2. If all Pods on a node are DaemonSet, the node is considered to be idle
