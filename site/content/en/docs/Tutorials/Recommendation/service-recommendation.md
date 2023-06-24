---
title: "Service Recommendation"
description: "Introduce for Service Recommendation"
weight: 16
---

Service recommendation scans the running status of Services in the cluster to help users find idle Kubernetes Services.

## Motivation

In Kubernetes, we usually use Service + Workload to automatically create and manage load balancing and attach it to applications. 
However, in daily operations, idle and low utilization load balancing inevitably occur, wasting a lot of costs. 
Service recommendation tries to help users find these Services to achieve cost optimization.

## Sample

```yaml
apiVersion: analysis.crane.io/v1alpha1
kind: Recommendation
metadata:
  annotations:
    analysis.crane.io/last-start-time: "2023-06-12 11:52:23"
    analysis.crane.io/message: Success
    analysis.crane.io/run-number: "7823"
  creationTimestamp: "2023-06-12T09:44:23Z"
  labels:
    analysis.crane.io/recommendation-rule-name: service-rule
    analysis.crane.io/recommendation-rule-recommender: Service
    analysis.crane.io/recommendation-rule-uid: 67807cd9-b4c9-4d63-8493-d330ccace364
    analysis.crane.io/recommendation-target-kind: Service
    analysis.crane.io/recommendation-target-name: nginx
    analysis.crane.io/recommendation-target-namespace: crane-system
    analysis.crane.io/recommendation-target-version: v1
  name: service-rule-service-cnwt5
  namespace: crane-system
  ownerReferences:
    - apiVersion: analysis.crane.io/v1alpha1
      blockOwnerDeletion: false
      controller: false
      kind: RecommendationRule
      name: service-rule
      uid: 67807cd9-b4c9-4d63-8493-d330ccace364
spec:
  adoptionType: StatusAndAnnotation
  completionStrategy:
    completionStrategyType: Once
  targetRef:
    apiVersion: v1
    kind: Service
    name: nginx
    namespace: crane-system
  type: Service
status:
  action: Delete
  description: It is a Orphan Service, Pod count is 0
  lastUpdateTime: "2023-06-12T11:52:23Z"
```

In this sample:

- The recommended TargetRef points to the Service: nginx.
- The recommendation type is Service recommendation.
- The action is to Delete, and it is only a suggestion provided here.

## Implement

Service recommendation completes a recommendation process using the following steps:

1. Scan all LoadBalancer-type Services in the cluster.
2. If the endpoints corresponding to the Service have an Address or NotReadyAddresses, it is not a restricted Service.
3. Based on the traffic-related metrics in Service recommendation, check whether the Service is below the threshold level. If it is below the threshold level, it is determined to be an idle node.


## How to verify the accuracy of recommendation results

The following is the Prom query for determining the threshold level of node resources. When verifying, replace "node" with the actual node name.

```go
// Container network cumulative count of bytes received
queryFmtNetReceiveBytes = `sum(rate(container_network_receive_bytes_total{namespace="%s",pod=~"%s",container!=""}[3m]))`
// Container network cumulative count of bytes transmitted
queryFmtNetTransferBytes = `sum(rate(container_network_transmit_bytes_total{namespace="%s",pod=~"%s",container!=""}[3m]))`
```

## Accepted resources

Only Service type is supported, and currently, only LoadBalancer-type Services will be analyzed.

## Configuration

| Configuration items      | Default value | Description                          |
|----------|-----|--------------------------------------|
| net-receive-bytes | 0   | The amount of network request bytes received by the Pods corresponding to the Service, which is not checked by default. |
| net-receive-percentile  | 0.99 | The percentile used to calculate the amount of network requests received              |
| net-transfer-bytes | 0   | The amount of network request bytes transmitted by the Pods corresponding to the Service, which is not checked by default.  |
| net-transfer-percentile | 0.99    | The percentile used when calculating the amount of network requests transmitted.              |

Note that when a pod is configured with a liveness/readiness probe, the kubelet's probing will bring some container traffic, so the threshold for traffic needs to be set slightly higher. The configuration can be combined with specific monitoring data.

How to update recommendation configuration please refer to：[**Recommendation Framework**](/docs/tutorials/recommendation/recommendation-framework)