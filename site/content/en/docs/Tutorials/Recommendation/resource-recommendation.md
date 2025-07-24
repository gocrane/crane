---
title: "Resource Recommendation"
description: "Introduce for Resource Recommendation"
weight: 14
---

Kubernetes' users often config request and limit based on empirical values when creating application resources. Based on the resource recommendation algorithm, you can analyze the actual application usage and recommend more appropriate resource configurations. You can use the resource recommendation algorithm to improve the resource utilization of the cluster.

## Motivation

In Kubernetes, Request defines the minimum amount of resources required by Pod, Limit defines the maximum amount of resources capability for Pod , and workload's Utilization = Resource Usage / Request. There are two types of unreasonable resource utilization:

- Utilization too low: The Request is often set to a large value because user don't know how many resource specifications can meet application requirements and choose to make it higher. It leads to a lot of resource waste.
- Utilization too high: Due to the service pressure caused by peak traffic or incorrect resource configuration. If the CPU usage is too high, it might cause more delay. If the memory usage exceeds the Limit, the Container will be killed, which affects service stability.

The figure below shows a workload with low utilization, it has 30% of the resource wasted between the Pod's peak historical usage and its Request.

![Resource Waste](/images/resource-waste.jpg)

Resource recommendation attempts to reduce the complexity of how to know the fit request of workloads by analyzing the historical usage.

## Sample

A Resource recommendation sample yaml looks like below:

```yaml
kind: Recommendation
apiVersion: analysis.crane.io/v1alpha1
metadata:
  name: workloads-rule-resource-flzbv
  namespace: crane-system
  labels:
    analysis.crane.io/recommendation-rule-name: workloads-rule
    analysis.crane.io/recommendation-rule-recommender: Resource
    analysis.crane.io/recommendation-rule-uid: 18588495-f325-4873-b45a-7acfe9f1ba94
    analysis.crane.io/recommendation-target-kind: Deployment
    analysis.crane.io/recommendation-target-name: load-test
    analysis.crane.io/recommendation-target-version: v1
    app: craned
    app.kubernetes.io/instance: crane
    app.kubernetes.io/managed-by: Helm
    app.kubernetes.io/name: crane
    app.kubernetes.io/version: v0.7.0
    helm.sh/chart: crane-0.7.0
  ownerReferences:
    - apiVersion: analysis.crane.io/v1alpha1
      kind: RecommendationRule
      name: workloads-rule
      uid: 18588495-f325-4873-b45a-7acfe9f1ba94
      controller: false
      blockOwnerDeletion: false
spec:
  targetRef:
    kind: Deployment
    namespace: crane-system
    name: craned
    apiVersion: apps/v1
  type: Resource
  completionStrategy:
    completionStrategyType: Once
  adoptionType: StatusAndAnnotation
status:
  recommendedValue:
    resourceRequest:
      containers:
        - containerName: craned
          target:
            cpu: 150m
            memory: 256Mi
        - containerName: dashboard
          target:
            cpu: 150m
            memory: 256Mi
  targetRef: {}
  recommendedInfo: >-
    {"spec":{"template":{"spec":{"containers":[{"name":"craned","resources":{"requests":{"cpu":"150m","memory":"256Mi"}}},{"name":"dashboard","resources":{"requests":{"cpu":"150m","memory":"256Mi"}}}]}}}}
  currentInfo: >-
    {"spec":{"template":{"spec":{"containers":[{"name":"craned","resources":{"requests":{"cpu":"500m","memory":"512Mi"}}},{"name":"dashboard","resources":{"requests":{"cpu":"200m","memory":"256Mi"}}}]}}}}
  action: Patch
  conditions:
    - type: Ready
      status: 'True'
      lastTransitionTime: '2022-11-29T04:07:44Z'
      reason: RecommendationReady
      message: Recommendation is ready
  lastUpdateTime: '2022-11-30T03:07:49Z'
```

In this sample：

- Recommendation TargetRef point to a Deployment in namespace crane-system：craned
- Recommendation type is Replicas
- adoptionType is StatusAndAnnotation，indicated that put recommendation result in recommendation.status and Deployment 的 Annotation
- recommendedInfo shows the recommended requests for containers（recommendedValue is deprecated），currentInfo shows the current request for containers. The format is Json that can be updated for TargetRef by `Kubectl Patch`

How to create a Resource recommendation please refer to：[**Recommendation Framework**](/docs/tutorials/recommendation/recommendation-framework)

## Implement

The process for one Resource recommendation:

1. Query the historical CPU and Memory usage of the Workload for the past week by monitoring system.
2. Take the P99 percentile usage through VPA Histogram, and then multiply the amplification factor
3. OOM protection: If the container has a history of OOM events, it is recommended to increase the memory size based on the memory used when OOM happened.
4. Resource specifications: The recommended result is rounded up based on the specified pod specifications

To sum up, based on the historical resource usage, set the Request value to slightly higher than the historical maximum and consider the OOM and Pod specifications.

### VPA Algorithm

The core algorithm of resource recommendation is to recommend reasonable resource request based on historical resource consumption. we adopt the community VPA Histogram algorithm to implement it. VPA algorithm puts the historical resource consumption into the histogram, finds the P99 percentage of resource consumption, and multiplies the percentage by the amplification factor as the recommended value.

The output of VPA algorithm is the P99 consumption of cpu and memory. In order to reserve buffer for the application, it will multiply the magnification factor. You can configure the amplification factor in either of the following ways:

1. Margin fraction: Recommended result = P99 usage * (1 + margin fraction), corresponding configuration: cpu-request-margin-fraction and me-request-margin-fraction
2. Target utilization: The recommended result is P99 amount/target utilization, and the corresponding configurations are cpu-target-utilization and mem-target-utilization

When you have a target peak utilization target for the application, it is recommended to use the **target utilization** way to amplify the recommendation results.

### OOM Protection

Craned runs a component, OOMRecorder, which records the events of the container OOM in the cluster. The resource recommendation reads the events of the OOM Recorder to obtain the memory usage at OOM time. We make sure the recommended memory is larger than the value when OOM happened. 

### Resource Specification

In Serverless Kubernetes, the cpu and memory specifications of the Pod are predefined. The resource recommendation can be rounded up according to the predefined resource specifications. For example, the recommended cpu value based on the historical usage is 0.125 core, and are rounded up to 0.25 core. You can also modify the specifications to meet the specifications requirements of your environment.

### Prometheus Metric 

Record recommended resource to Metric：crane_analysis_resource_recommendation

## How to verify the accuracy of recommendation results

You can use the following Prom-query to obtain the Workload Container resource usage. The recommended value is slightly higher than the historical maximum, considering the OOM and Pod specifications.

Taking Deployment Craned in crane-system as an example, you can use your container, namespace to replace it in following Prom-query.

```shell
irate(container_cpu_usage_seconds_total{container!="POD",namespace="crane-system",pod=~"^craned.*$",container="craned"}[3m])   # cpu usage
```

```shell
container_memory_working_set_bytes{container!="POD",namespace="crane-system",pod=~"^craned.*$",container="craned"}  # memory usage
```

## Accepted Resources

Support StatefulSet and Deployment by default，but all workloads that support `Scale SubResource` are supported.

## Configuration

| Configuration items         | Default      | Description                                                                                       |
|-----------------------------|--------------|---------------------------------------------------------------------------------------------------|
| cpu-sample-interval         | 1m           | Metric sampling interval for requesting CPU monitoring data                                       |
| cpu-request-percentile      | 0.99         | Target CPU Percentile that used for VPA                                                           |
| cpu-request-margin-fraction | 0.15         | CPU recommend value margin factor，0.15 means recommended value = recommended value * 1.15         |
| cpu-target-utilization      | 1            | CPU target utilization，0.8 means recommended value = recommended value / 0.8                      |
| cpu-model-history-length    | 168h         | Historical length for CPU monitoring data                                                         |
| mem-sample-interval         | 1m           | Metric sampling interval for requesting Memory monitoring data                                    |
| mem-request-percentile      | 0.99         | Target Memory Percentile that used for VPA                                                        |
| mem-request-margin-fraction | 0.15         | Memory recommend value margin factor，0.15 means recommended value = recommended value * 1.15      |
| mem-target-utilization      | 1            | Memory target utilization，0.8 means recommended value = recommended value / 0.8                   |
| mem-model-history-length    | 168h         | Historical length for Memory monitoring data                                                      |
| specification               | false        | Enable for resource rpecification                                                                 |
| specification-config        | ""           | resource specifications configuration                                                             |
| oom-protection              | true         | Enable for OOM Prodection                                                                         |
| oom-history-length          | 168h         | OOM event history length, ignore too old events                                                   |
| oom-bump-ratio              | 1.2          | OOM memory bump up ratio                                                                          |
| cpu-histogram-bucket-size   | 0.1          | The size of the balanced bucket is also equal to the minimum value recommended by the CPU         |
| cpu-histogram-max-value     | 100          | The maximum value of the balance bucket is also equal to the maximum value recommended by the CPU |
| mem-histogram-bucket-size   | 104857600    | The size of the balanced bucket is also equal to the minimum value recommended by the MEM         |
| mem-histogram-max-value     | 104857600000 | The maximum value of the balance bucket is also equal to the maximum value recommended by the MEM |

How to update recommendation configuration please refer to：[**Recommendation Framework**](/docs/tutorials/recommendation/recommendation-framework)

## Default resource specifications configuration

| CPU（Cores） | Memory（GBi） |
|------------|-------------|
| 0.25       | 0.25        |
| 0.25       | 0.5         |
| 0.25       | 1           |
| 0.5        | 0.5         |
| 0.5        | 1           |
| 1          | 1           |
| 1          | 2           |
| 1          | 4           |
| 1          | 8           |
| 2          | 2           |
| 2          | 4           |
| 2          | 8           |
| 2          | 16          |
| 4          | 4           |
| 4          | 8           |
| 4          | 16          |
| 4          | 32          |
| 8          | 8           |
| 8          | 16          |
| 8          | 32          |
| 8          | 64          |
| 16         | 32          |
| 16         | 64          |
| 16         | 128         |
| 32         | 64          |
| 32         | 128         |
| 32         | 256         |
| 64         | 128         |
| 64         | 256         |
