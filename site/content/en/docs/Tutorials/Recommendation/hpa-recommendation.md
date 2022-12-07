---
title: "HPA Recommendation（Alpha）"
description: "Introduce for HPA Recommendation"
weight: 16
---

Kubernetes' users want to use HPA to optimize resource utilization. But it is often that we don't know which applications are suitable for HPA or how to configure the parameters of HPA. With HPA Recommendation you can analyze the actual application usage and get recommended configurations. You can use it to improve application resource utilization.

HPA recommendation is still in Alpha phase, comments are welcome.

## Motivation

In Kubernetes, the HPA (HorizontalPodAutoscaler) automatically updates the workload replicas (such as Deployment or StatefulSet) to meet the target utilization. However, in the actual world, we observe following problems:

- Some applications should improve resource utilization through HPA, but HPA is not configured
- Some HPA configuration is not reasonable, can not effectively perform autoscaling, also can not improve resource utilization.

Based on the historical metrics data and algorithm analysis, HPA Recommendation provide the following suggestions: Which applications are suitable for HPA and How to configure it.

## Sample

An HPA recommendation sample yaml looks like below:

```yaml
apiVersion: analysis.crane.io/v1alpha1
kind: Recommendation
metadata:
  labels:
    analysis.crane.io/recommendation-rule-name: workload-hpa
    analysis.crane.io/recommendation-rule-recommender: HPA
    analysis.crane.io/recommendation-rule-uid: 0214c84b-8b39-499b-a7c6-559ac460695d
    analysis.crane.io/recommendation-target-kind: Rollout
    analysis.crane.io/recommendation-target-name: eshop
    analysis.crane.io/recommendation-target-version: v1alpha1
  name: workload-hpa-hpa-blr4r
  namespace: zytms
  ownerReferences:
    - apiVersion: analysis.crane.io/v1alpha1
      blockOwnerDeletion: false
      controller: false
      kind: RecommendationRule
      name: workload-hpa
      uid: 0214c84b-8b39-499b-a7c6-559ac460695d
spec:
  adoptionType: StatusAndAnnotation
  completionStrategy:
    completionStrategyType: Once
  targetRef:
    apiVersion: argoproj.io/v1alpha1
    kind: Rollout
    name: eshop
    namespace: eshop
  type: HPA
status:
  action: Create
  lastUpdateTime: "2022-12-05T06:12:54Z"
  recommendedInfo: '{"kind":"EffectiveHorizontalPodAutoscaler","apiVersion":"autoscaling.crane.io/v1alpha1","metadata":{"name":"eshop","namespace":"eshop","creationTimestamp":null},"spec":{"scaleTargetRef":{"kind":"Rollout","name":"eshop","apiVersion":"argoproj.io/v1alpha1"},"minReplicas":1,"maxReplicas":1,"scaleStrategy":"Preview","metrics":[{"type":"Resource","resource":{"name":"cpu","target":{"type":"Utilization","averageUtilization":58}}},{"type":"Pods","pods":{"metric":{"name":"k8s_pod_cpu_core_used"},"target":{"type":"AverageValue","averageValue":"500m"}}}]},"status":{}}'
  recommendedValue: |
    effectiveHPA:
      maxReplicas: 1
      metrics:
      - resource:
          name: cpu
          target:
            averageUtilization: 58
            type: Utilization
        type: Resource
      - pods:
          metric:
            name: k8s_pod_cpu_core_used
          target:
            averageValue: 500m
            type: AverageValue
        type: Pods
      minReplicas: 1
```

In this sample：

- Recommendation TargetRef point to a ArgoRollout in eshop namespace: eshop
- Recommendation type is HPA
- adoptionType is StatusAndAnnotation，indicated that put recommendation result in recommendation.status and Workload's Annotation
- recommendedInfo shows the recommended HPA configuration.（recommendedValue is deprecated）
- action is Create，If existing EHPA in k8s cluster, then the action will be Patch 

## Implement

The process for one HPA recommendation:

1. Query the historical CPU and Memory usage of the Workload for the past week by monitoring system.
2. Use DSP algorithm to predict the CPU usage in the future.
3. Calculate the replicas for both CPU and memory, then choose a larger one.
4. Calculate the historical CPU usage fluctuation and minimum usage, and filter out the suitable Workload for HPA
5. Calculate the targetUtilization based on the peak CPU utilization of the pod
6. Calculate the recommended maxReplicas based on the recommended targetUtilization
7. Assemble the targetUtilization, maxReplicas, and minReplicas into a complete EHPA object as recommended result

### How to filter the suitable workload suitable for HPA

It should meet the following conditions:

1. The Workload is healthy. For example, most of the Pods are running
2. There are peaks and troughs in CPU usage. A fixed number of replicas is recommended if the usage is largely smooth and steady or completely random
3. Workload with a certain amount cpu usage. If the cpu usage is very low for a long time, HPA is no need even if there is some fluctuation

The following is a typical workload with peaks and troughs which is suitable for HPA.

![](/images/algorithm/dsp/input0.png)

### Algorithm for recommend MinReplicas

The method is consistent with replica recommendation. Please referring to: [**Replicas Recommendation**](/docs/tutorials/recommendation/replicas-recommendation)

## Accepted resources

Support StatefulSet and Deployment by default，but all workloads that support `Scale SubResource` are supported.

## Configuration

| Configuration items   | Default | Description                                                                       |
|-------------|---------|-----------------------------------------------------------------------------------|
| workload-min-replicas | 1       | Workload replicas that less than this value will abort recommendation             |
| pod-min-ready-seconds | 30      | Defines the min seconds to identify Pod is ready                                  |
| pod-available-ratio | 0.5     | Workload ready Pod ratio that less than this value will abort recommendation      |
| default-min-replicas | 1       | default minReplicas                                                               |
| cpu-percentile | 0.95    | Percentile for historical cpu usage                                               |
| mem-percentile | 0.95    | Percentile for historical memory usage                                            |
| cpu-target-utilization | 0.5     | Target of CPU peak historical usage                                               |
| mem-target-utilization | 0.5     | Target of Memory peak historical usage                                            |
| predictable | false   | When set to true, it will not recommend for HPA if CPU usage is not predictable   |
| reference-hpa | true    | The recommended result will inherits custom/external metric from existing ehpa    |
| min-cpu-usage-threshold | 1       | Workload CPU peak usage that less than this value will abort recommendation       |
| fluctuation-threshold | 1.5     | Workload CPU usage fluctuation that less than this value will abort recommendation |
| min-cpu-target-utilization | 30      | minimum CPU TargetUtilization                                                     |
| max-cpu-target-utilization | 75      | maximum CPU TargetUtilization                                                     |
| max-replicas-factor | 3       | the factor when calculate maxReplicas                                       |

How to update recommendation configuration please refer to：[**Recommendation Framework**](/docs/tutorials/recommendation/recommendation-framework)
