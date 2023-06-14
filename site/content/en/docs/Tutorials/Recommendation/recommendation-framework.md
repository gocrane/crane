---
title: "Recommendation Framework"
description: "Introduce for Recommendation Framework and How to use it"
weight: 10
---

**Recommendation Framework** provides the ability to automatically analyze various resources in Kubernetes cluster and make recommendations for optimization。

## Overview

Crane's recommendation module periodically detects cluster resource configuration problems and provides optimization suggestions. Framework provides a variety of recommender to implement the optimization and recommendation for different resources.
If you want to know how Crane makes Recommendations, or if you want to try to implement a custom Recommender, or change the recommender's implements, this article will help you know How it does.

## Use Case

The following are typical use cases for recommendation framework：

- Create a RecommendationRule configuration. RecommendationRule Controller will conduct Recommendation tasks periodically based on the configuration and give recommendations.
- Adjust the allocation of resources according to the optimization Recommendation.

## Create RecommendationRule

This is a sample for RecommendationRule: workload-rule.yaml。

```yaml
apiVersion: analysis.crane.io/v1alpha1
kind: RecommendationRule
metadata:
  name: workloads-rule
spec:
  runInterval: 24h                            # Run every 24 hours
  resourceSelectors:                          # configuration for resources
    - kind: Deployment
      apiVersion: apps/v1
    - kind: StatefulSet
      apiVersion: apps/v1
  namespaceSelector:
    any: true                                 # scan all namespace
  recommenders:                               # Replicas and Resource recommenders for Workload 
    - name: Replicas
    - name: Resource
```

In this example：

- The analysis is run every 24 hours. `runInterval` format is an time interval, for example, 1h or 1m. If this parameter is set to null, the analysis is run only once.
- Resources to be analyzed are set by configuring the 'resourceSelectors' array. Each `resourceSelector` selects resources in k8s by kind, apiVersion, name. If no name is specified, it indicates all resources based on `namespaceSelector`.
- `namespaceSelector` defines the namespace of the resource to be analyzed. `any: true` indicates that all namespaces are selected
- `recommenders` define which recommender should be used to analyze the resources. Currently, supported types: [recommenders](/docs/tutorials/recommendation/recommendation-framework#recommender)
- the Resource type and ` recommenders ` need to match, such as the Resource recommended default only support Deployments and StatefulSets. Please refer to the Recommender docs to know which resources it supports

1. Create the RecommendationRule with the following command, a recommendation will start as soon as it is created.

```shell
kubectl apply -f workload-rules.yaml
```

This example will analysis Resource and Replicas for Deployments and StatefulSets in all namespace.。

2. Check the RecommendationRule recommendation progress. Observe the progress of recommendation tasks through `Status.recommendations`. Recommendation tasks are executed sequentially. If lastStartTime of all tasks is the latest time and message has value, it indicates that this recommendation is completed.

```shell
kubectl get rr workloads-rule
```

```yaml
status:
  lastUpdateTime: "2022-09-28T10:36:02Z"
  recommendations:
  - apiVersion: analysis.crane.io/v1alpha1
    kind: Recommendation
    lastStartTime: "2022-09-28T10:36:02Z"
    message: Success
    name: workloads-rule-replicas-rckvb
    namespace: default
    recommenderRef:
      name: Replicas
    targetRef:
      apiVersion: apps/v1
      kind: Deployment
      name: php-apache
      namespace: default
    uid: b15cbcd7-6fe2-4ace-9ae8-11cc0a6e69c2
  - apiVersion: analysis.crane.io/v1alpha1
    kind: Recommendation
    lastStartTime: "2022-09-28T10:36:02Z"
    message: Success
    name: workloads-rule-resource-pnnxn
    namespace: default
    recommenderRef:
      name: Resource
    targetRef:
      apiVersion: apps/v1
      kind: Deployment
      name: php-apache
      namespace: default
    uid: 8472013a-bda2-4025-b0df-3fdc69c1c910
```

3. Check `Recommendation`

You can filter `Recommendation` through these labels，like `kubectl get recommend -l analysis.crane.io/recommendation-rule-name=workloads-rule`

- RecommendationRule Name：analysis.crane.io/recommendation-rule-name
- RecommendationRule UID：analysis.crane.io/recommendation-rule-uid
- RecommendationRule recommender：analysis.crane.io/recommendation-rule-recommender
- Recommended resource's kind：analysis.crane.io/recommendation-target-kind
- Recommended resource's version：analysis.crane.io/recommendation-target-version
- Recommended resource's name：analysis.crane.io/recommendation-target-name

In general, the namespace of `Recommendation` is equal to the namespace of the recommended resource. But Recommendation for idle nodes is excluded, which are in the root namespace of crane, and the default root namespace is Crane-system.

## Adjust the allocation of resources according to the optimization Recommendation

For resource/replicas recommendation and recommendedInfo, users can PATCH status.recommendedinfo to workload to update the resource configuration, for example:

```shell
patchData=`kubectl get recommend workloads-rule-replicas-rckvb -n default -o jsonpath='{.status.recommendedInfo}'`;kubectl patch Deployment php-apache -n default --patch "${patchData}"
```

For idle nodes, users can offline or reduce the capacity of idle nodes based on their requirements.

## Recommender

Currently, Crane support these Recommenders:

- [**Resource Recommendation**](/docs/tutorials/recommendation/resource-recommendation): Use the VPA algorithm to analyze the actual usage of applications and recommend more appropriate resource configurations.
- [**Replicas Recommendation**](/docs/tutorials/recommendation/replicas-recommendation): Use the HPA algorithm to analyze the actual usage of applications and recommend more appropriate replicas configurations.
- [**HPA Recommendation**](/docs/tutorials/recommendation/hpa-recommendation): Scan the Workload in a cluster and recommend HPA configurations for Workload that are suitable for horizontal autoscaling
- [**IdleNode Recommendation**](/docs/tutorials/recommendation/idlenode-recommendation): Find the idle nodes in cluster
- [**Service Recommendation**](/zh-cn/docs/tutorials/recommendation/service-recommendation): Find the idle load balancer service in cluster
- [**PV Recommendation**](/zh-cn/docs/tutorials/recommendation/pv-recommendation): Find the idle persist volume in cluster

### Recommender Framework 

Recommender framework defines a set of workflow, The workflow execution sequence according to the process, the process is divided into four stages: Filter, Prepare, Recommend, Observe. Recommender performs recommends functions by implementing these four stages.

If you want to implement or extend a Recommender, please refer to :[How to develop recommender](/docs/tutorials/recommendation/how-to-develop-recommender)

## RecommendationConfiguration

RecommendationConfiguration defines the configuration for recommender. It will deploy a ConfigMap in crane root namespace: recommendation-configuration.

The following is a sample for RecommendationConfiguration.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: recommendation-configuration
  namespace: crane-system
data:
  config.yaml: |-
    apiVersion: analysis.crane.io/v1alpha1
    kind: RecommendationConfiguration
    recommenders:
      - name: Replicas
        acceptedResources:                   # Accepted resources
          - kind: Deployment
            apiVersion: apps/v1
          - kind: StatefulSet
            apiVersion: apps/v1
        config:                              # settings for recommender
          workload-min-replicas: "1"         
      - name: Resource
        acceptedResources:                   # Accepted resources
          - kind: Deployment
            apiVersion: apps/v1
          - kind: StatefulSet
            apiVersion: apps/v1
```

Users can modify ConfigMap and rolling update Crane to make new configuration works.

## How to make recommendations more accurate

The more historical data stored in a monitoring system, such as Prometheus, the more accurate the recommendation will be. More than two weeks's data is recommended for production. Predictions about new apps are often inaccurate.
