---
title: "Replicas Recommendation"
description: "Introduce for Replicas Recommendation"
weight: 13
---

Kubernetes' users often set the replicas based on empirical values when creating application resources. Based on the replicas recommendation, you can analyze the actual application usage and recommend a more suitable replicas configuration. You can use it to improve the resource utilization of the cluster.

## Implement

Based on the historical Workload CPU loads, find the workload's lowest CPU usage per hour in the past seven days, and calculate the replicas with 50% (configurable) cpu usage that should be configured

### Filter Phase

1. workload with low replicas: If the replicas is too low, it may not have high recommendation demand. Associated configuration: 'workload-min-replicas'
2. There is a certain percentage of the not running pods for workload: if the Pod of workload mostly can't run normally, may not be suitable for recommendation, associated configuration: `pod-min-ready-seconds` | `pod-available-ratio`

### Prepare Phase

Query the workload cpu usage in the past week.

### Recommend Phase

1. Calculate the lowest value of the median workload usage per hour in the past seven days (to prevent the impact of the minimum value): workload_cpu_usage_medium_min
2. The number of replicas corresponding to the target utilization:

```go
   	replicas := int32(math.Ceil(workload_cpu_usage_medium_min / (rr.TargetUtilization * float64(requestTotal) / 1000.)))
```

3. In order to prevent too low replicas，replicas should be larger than or equal to default-min-replicas

### Observe Phase

Record recommended replicas to Metric: crane_analytics_replicas_recommendation

## Accepted resources

Support StatefulSet and Deployment by default，but all workloads that support `Scale SubResource` are supported.

## Configuration

| Configuration items    | Default | Description                                                         |
|------------------------|---------|---------------------------------------------------------------------|
| workload-min-replicas  | 1       | Workload replicas than less than this value are not recommended     |
| pod-min-ready-seconds  | 30      | Defines the min seconds to identify Pod is ready                    |
| pod-available-ratio    | 0.5     | Workload ready Pod ratio that less than this value are not recommended |
| default-min-replicas   | 1       | default minReplicas                                                 |
| cpu-target-utilization | 0.5     | Calculate the minimum replicas based on this cpu utilization      |
