# HPA Recommendation

Using hpa recommendations, you can find resources in the cluster that are suitable for autoscaling, and use Crane's recommended result to create autoscaling object: [Effective HorizontalPodAutoscaler](tutorials/using-time-series-prediction.md)

## Create HPA Analytics

Create an **Resource** `Analytics` to give recommendation for deployment: `nginx-deployment` as a sample.

```bash
kubectl apply -f https://raw.githubusercontent.com/gocrane/crane/main/examples/analytics/nginx-deployment.yaml
kubectl apply -f https://raw.githubusercontent.com/gocrane/crane/main/examples/analytics/analytics-hpa.yaml
kubectl get analytics
```

```yaml title="analytics-hpa.yaml"
apiVersion: analysis.crane.io/v1alpha1
kind: Analytics
metadata:
  name: nginx-hpa
spec:
  type: HPA                        # This can only be "Resource" or "HPA".
  completionStrategy:
    completionStrategyType: Periodical  # This can only be "Once" or "Periodical".
    periodSeconds: 600                  # analytics selected resources every 10 minutes
  resourceSelectors:                    # defines all the resources to be select with
    - kind: Deployment
      apiVersion: apps/v1
      name: nginx-deployment
  config:                               # defines all the configuration for this analytics
    ehpa.deployment-min-replicas: "1"
    ehpa.fluctuation-threshold: "0"
    ehpa.min-cpu-usage-threshold: "0"
```

The output is:

```bash
NAME        AGE
nginx-hpa   16m
```

You can get created recommendation from analytics status:

```bash
kubectl get analytics nginx-hpa -o yaml
```

The output is similar to:

```yaml
apiVersion: analysis.crane.io/v1alpha1
kind: Analytics
metadata:
  creationTimestamp: "2022-05-15T13:34:19Z"
  name: nginx-hpa
  namespace: default
spec:
  completionStrategy:
    completionStrategyType: Periodical
    periodSeconds: 600
  config:
    ehpa.deployment-min-replicas: "1"
    ehpa.fluctuation-threshold: "0"
    ehpa.min-cpu-usage-threshold: "0"
  resourceSelectors:
  - apiVersion: apps/v1
    kind: Deployment
    labelSelector: {}
    name: nginx-deployment
  type: HPA
status:
  conditions:
  - lastTransitionTime: "2022-05-15T13:34:19Z"
    message: Analytics is ready
    reason: AnalyticsReady
    status: "True"
    type: Ready
  lastUpdateTime: "2022-05-15T13:34:19Z"
  recommendations:
  - lastStartTime: "2022-05-15T13:34:19Z"
    message: Success
    name: nginx-hpa-hpa-cd86s
    namespace: default
    targetRef:
      apiVersion: apps/v1
      kind: Deployment
      name: nginx-deployment
      namespace: default
    uid: b3cea8cb-259d-4cb2-bbbe-cd0e6544daaf
```

## Recommendation: Analytics result

The recommendation name presents on `status.recommendations[0].name`. Then you can get recommendation detail by running:

```bash
kubectl get recommend nginx-hpa-hpa-cd86s -o yaml
```

The output is similar to:

```yaml
apiVersion: analysis.crane.io/v1alpha1
kind: Recommendation
metadata:
  creationTimestamp: "2022-05-15T13:34:19Z"
  generateName: nginx-hpa-hpa-
  generation: 2
  labels:
    analysis.crane.io/analytics-name: nginx-hpa
    analysis.crane.io/analytics-type: HPA
    analysis.crane.io/analytics-uid: 5564edd0-d7cd-4da6-865b-27fa4fddf7c4
    app: nginx
  name: nginx-hpa-hpa-cd86s
  namespace: default
  ownerReferences:
  - apiVersion: analysis.crane.io/v1alpha1
    blockOwnerDeletion: false
    controller: false
    kind: Analytics
    name: nginx-hpa
    uid: 5564edd0-d7cd-4da6-865b-27fa4fddf7c4
spec:
  adoptionType: StatusAndAnnotation
  completionStrategy:
    completionStrategyType: Once
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: nginx-deployment
    namespace: default
  type: HPA
status:
  conditions:
  - lastTransitionTime: "2022-05-15T13:34:19Z"
    message: Recommendation is ready
    reason: RecommendationReady
    status: "True"
    type: Ready
  lastUpdateTime: "2022-05-15T13:34:19Z"
  recommendedValue: |
    maxReplicas: 2
    metrics:
    - resource:
        name: cpu
        target:
          averageUtilization: 75
          type: Utilization
      type: Resource
    minReplicas: 2
```

## HPA Recommendation Algorithm model

### Inspecting

1. Workload with low replicas: If the replicas is too low,  may not be suitable for hpa recommendation. Associated configuration: ehpa.deployment-min-replicas | ehpa.statefulset-min-replicas | ehpa.workload-min-replicas
2. Workload with a certain percentage of not running pods: if the workload of Pod mostly can't run normally, may not be suitable for flexibility. Associated configuration: ehpa.pod-min-ready-seconds | ehpa.pod-available-ratio
3. Workload with low cpu usage: The low CPU usage workload means that there is no load pressure. In this case, we can't estimate it. Associated configuration: ehpa.min-cpu-usage-threshold
4. Workload with low fluctuation of cpu usage: dividing of the maximum and minimum usage is defined as the fluctuation rate. If the fluctuation rate is too low, the workload will not benefit much from hpa. Associated configuration: ehpa.fluctuation-threshold 

### Advising

In the advising phase, one EffectiveHPA Spec is recommended using the following Algorithm model. The recommended logic for each field is as follows:

**Recommend TargetUtilization**

Principle: Use Pod P99 resource utilization to recommend hpa. Because if the application can accept this utilization over P99 time, it can be inferred as a target for elasticity.

1. Get the Pod P99 usage of the past seven days by Percentile algorithm: pod_cpu_usage_p99
2. Corresponding utilization: target_pod_CPU_utilization = pod_cpu_usage_p99 / pod_cpu_request
3. To prevent over-utilization or under-utilization, target_pod_cpu_utilization needs to be less than ehpa.min-cpu-target-utilization and greater than ehpa. max-cpu-target-utilization

**Recommend minReplicas**

Principle: MinReplicas are recommended for the lowest hourly workload utilization for the past seven days.

1. Calculate the lowest median workload cpu usage of the past seven days: workload_cpu_usage_medium_min
2. Corresponding replicas: minReplicas = workload_cpu_usage_medium_min / pod_cpu_request / ehpa.max-cpu-target-utilization
3. To prevent the minReplicas being too small, the minReplicas must be greater than or equal to ehpa.default-min-replicas

**Recommend maxReplicas**

Principle: Use workload's past and future seven days load to recommend maximum replicas.

1. Calculate P95 workload CPU usage for the past seven days and the next seven days: workload_cpu_usage_p95
2. Corresponding replicas: max_replicas_origin = workload_cpu_usage_p95 / pod_cpu_request / target_cpu_utilization
3. To handle with the peak traffic, Magnify by a certain factor: max_replicas = max_replicas_origin * ehpa.max-replicas-factor

**Recommend MetricSpec(except CpuUtilization)**

1. If HPA is configured for workload, MetricSpecs other than CpuUtilization are inherited

**Recommend Behavior**

1. If HPA is configured for workload, the corresponding Behavior configuration is inherited

**Recommend Prediction**

1. Try to predict the CPU usage of the workload in the next seven days using DSP
2. If the prediction is successful, add the prediction configuration
3. If the workload is not predictable, do not add the prediction configuration.

## Configurations for HPA Recommendation

- ehpa.deployment-min-replicas: the default value is 1，hpa recommendations are not made for workloads smaller than this value
- ehpa.statefulset-min-replicas: the default value is 1，hpa recommendations are not made for workloads smaller than this value
- ehpa.workload-min-replicas: the default value is 1，Workload replicas smaller than this value are not recommended for hpa.
- ehpa.pod-min-ready-seconds: the default value is 30，specifies the number of seconds in decide whether a POD is ready.
- ehpa.pod-available-ratio: the default value is 0.5，Workloads whose Ready pod ratio is smaller than this value are not recommended for hpa.
- ehpa.default-min-replicas: the default value is 2，the default minimum minReplicas.
- ehpa.max-replicas-factor: the default value is 3，the factor for calculate maxReplicas.
- ehpa.min-cpu-usage-threshold: the default value is 10, hpa recommendations are not made for workloads smaller than this value.
- ehpa.fluctuation-threshold: the default value is 1.5, hpa recommendations are not made for workloads smaller than this value.
- ehpa.min-cpu-target-utilization: the default value is 30
- ehpa.max-cpu-target-utilization: the default value is 75
- ehpa.reference-hpa: the default value is true, which means inherits the existing HPA configuration
