---
title: "PV Recommendation"
description: "Introduce for PV Recommendation"
weight: 17
---

PV recommendation scans the running status of PVs in the cluster to help users find idle Kubernetes PVs.

## Motivation

In Kubernetes, we usually use PV + Workload to automatically create and manage storage volumes and attach them to applications. 
However, in daily operations, idle or unused storage volumes may inevitably occur, wasting a lot of costs. 
PV recommendation tries to help users find these PVs to achieve cost optimization.

## Sample

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

In this sample:

- The recommended TargetRef points to PV: pvc-6ce24277-24e9-4fcf-8e8a-f9bdb5694134
- The recommendation type is PV recommendation
- The action is to Delete, and it is only a suggestion provided here.

## Implement

PV recommendation completes a recommendation process using the following steps:

1. Scan all PVs in the cluster and find the list of Pods corresponding to each PV.
2. If the PV does not have a corresponding PVC, it is considered an idle PV.
3. If no Pods are associated with this PV and PVC, it is considered an idle PVC.

## Configuration

Currently, there is no parameter configuration for PV recommendation.

How to update recommendation configuration please refer to：[**Recommendation Framework**](/docs/tutorials/recommendation/recommendation-framework)