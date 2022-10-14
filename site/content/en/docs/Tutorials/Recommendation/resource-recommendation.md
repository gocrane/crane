---
title: "Resource Recommendation"
description: "Introduce for Resource Recommendation"
weight: 14
---

Kubernetes' users often config request and limit based on empirical values when creating application resources. Based on the resource recommendation algorithm, you can analyze the actual application usage and recommend more appropriate resource configurations. You can use the resource recommendation algorithm to improve the resource utilization of the cluster.

## Implement

The algorithm model adopts VPA's Moving Window algorithm for recommendation

1. By monitoring system, you can obtain the Workload (configurable) CPU and Memory usage history in the past week. 
2. The algorithm considers the timeliness of data. The newer data sampling points will have higher weight.
3. The recommended CPU value is calculated based on the target percentile value that set by the user, and the recommended Memory value is calculated based on the maximum historical value

### Filter Phase

Workloads that have no Pods: If the workload does not have Pods, analysis cannot be performed

### Recommend Phase

Adopt VPA Moving Window algorithm to calculate CPU and Memory for every container and give recommendation config.

### Observe Phase

Record recommended resource to Metric：crane_analytics_replicas_recommendation

## Accepted Resources

Support StatefulSet and Deployment by default，but all workloads that support `Scale SubResource` are supported.

## Configuration

| Configuration items    | Default | Description                                                                                  |
|-----------------------------|------|----------------------------------------------------------------------------------------------|
| cpu-sample-interval         | 1m   | Metric sampling interval for requesting CPU monitoring data                                  |
| cpu-request-percentile      | 0.99 | Target CPU Percentile that used for VPA                                                      |
| cpu-request-margin-fraction | 0.15 | CPU recommend value margin factor，0.15 means recommended value = recommended value * 1.15    |
| cpu-target-utilization      | 1    | CPU target utilization，0.8 means recommended value = recommended value / 0.8                 |
| cpu-model-history-length    | 168h | Historical length for CPU monitoring data                                                    |
| mem-sample-interval         | 1m   | Metric sampling interval for requesting Memory monitoring data                               |
| mem-request-percentile      | 0.99 | Target Memory Percentile that used for VPA                                                   |
| mem-request-margin-fraction | 0.15 | Memory recommend value margin factor，0.15 means recommended value = recommended value * 1.15 |
| mem-target-utilization      | 1    | Memory target utilization，0.8 means recommended value = recommended value / 0.8              |
| mem-model-history-length    | 168h | Historical length for Memory monitoring data                                                 |
