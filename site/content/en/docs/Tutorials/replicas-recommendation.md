---
title: "Replicas Recommendation"
description: "How to use Replicas Recommendation"
weight: 13
---

Kubernetes' users often set the replicas of workload or HPA configurations based on empirical values. Replicas recommendation analyze the actual application usage and give advice for replicas and HPA configurations. You can refer to and adopt it for your workloads to improve cluster resource utilization.

## Features

1. Algorithm: The algorithm for calculating the replicas refers to HPA, and supports to customization algo args
2. HPA recommendations: Scan for applications that suitable for configuring horizontal elasticity (EHPA), And give advice for configuration of EHPA, [EHPA](/docs/tutorials/using-effective-hpa-to-scaling-with-effectiveness) is a smart horizontal elastic product provided by Crane
3. Support batch analysis: With the ResourceSelector, users can batch analyze multiple workloads

## Create HPA Analytics

Create an **Resource** `Analytics` to give recommendation for deployment: `nginx-deployment` as a sample.

=== "Main"

      ```bash
      kubectl apply -f https://raw.githubusercontent.com/gocrane/crane/main/examples/analytics/nginx-deployment.yaml
      kubectl apply -f https://raw.githubusercontent.com/gocrane/crane/main/examples/analytics/analytics-replicas.yaml
      ```

=== "Mirror"

      ```bash
      kubectl apply -f https://gitee.com/finops/crane/raw/main/examples/analytics/nginx-deployment.yaml
      kubectl apply -f https://gitee.com/finops/crane/raw/main/examples/analytics/analytics-replicas.yaml
      ```

The created `Analytics` yaml is following:

```yaml title="analytics-replicas.yaml"
apiVersion: analysis.crane.io/v1alpha1
kind: Analytics
metadata:
  name: nginx-hpa
spec:
  type: Replicas                        # This can only be "Resource" or "Replicas".
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

You can get created recommendations from analytics status:

```bash
kubectl get analytics nginx-replicas -o yaml
```

The output is similar to:

```yaml
apiVersion: analysis.crane.io/v1alpha1
kind: Analytics
metadata:
  name: nginx-replicas
  namespace: default
spec:
  completionStrategy:
    completionStrategyType: Periodical
    periodSeconds: 600
  config:
    replicas.fluctuation-threshold: "0"
    replicas.min-cpu-usage-threshold: "0"
    replicas.workload-min-replicas: "1"
  resourceSelectors:
  - apiVersion: apps/v1
    kind: Deployment
    labelSelector: {}
    name: nginx-deployment
  type: Replicas
status:
  conditions:
  - lastTransitionTime: "2022-06-02T09:44:54Z"
    message: Analytics is ready
    reason: AnalyticsReady
    status: "True"
    type: Ready
  lastUpdateTime: "2022-06-02T09:44:54Z"
  recommendations:
  - lastStartTime: "2022-06-02T09:44:54Z"
    message: Success
    name: nginx-replicas-replicas-7qspm
    namespace: default
    targetRef:
      apiVersion: apps/v1
      kind: Deployment
      name: nginx-deployment
      namespace: default
    uid: c853043c-5ff6-4ee0-a941-e04c8ec3093b
```

## Recommendation: Analytics result

Use label selector to get related recommendations owns by `Analytics`.

```bash
kubectl get recommend -l analysis.crane.io/analytics-name=nginx-replicas -o yaml
```

The output is similar to:

```yaml
apiVersion: v1
items:
   - apiVersion: analysis.crane.io/v1alpha1
     kind: Recommendation
     metadata:
        creationTimestamp: "2022-06-02T09:44:54Z"
        generateName: nginx-replicas-replicas-
        generation: 2
        labels:
           analysis.crane.io/analytics-name: nginx-replicas
           analysis.crane.io/analytics-type: Replicas
           analysis.crane.io/analytics-uid: e9168c6e-329f-40e9-8d0f-a1ddc35b0d47
           app: nginx
        name: nginx-replicas-replicas-7qspm
        namespace: default
        ownerReferences:
           - apiVersion: analysis.crane.io/v1alpha1
             blockOwnerDeletion: false
             controller: false
             kind: Analytics
             name: nginx-replicas
             uid: e9168c6e-329f-40e9-8d0f-a1ddc35b0d47
        resourceVersion: "818959913"
        selfLink: /apis/analysis.crane.io/v1alpha1/namespaces/default/recommendations/nginx-replicas-replicas-7qspm
        uid: c853043c-5ff6-4ee0-a941-e04c8ec3093b
     spec:
        adoptionType: StatusAndAnnotation
        completionStrategy:
           completionStrategyType: Once
        targetRef:
           apiVersion: apps/v1
           kind: Deployment
           name: nginx-deployment
           namespace: default
        type: Replicas
     status:
        conditions:
           - lastTransitionTime: "2022-06-02T09:44:54Z"
             message: Recommendation is ready
             reason: RecommendationReady
             status: "True"
             type: Ready
        lastUpdateTime: "2022-06-02T09:44:54Z"
        recommendedValue: |
           effectiveHPA:
             maxReplicas: 3
             metrics:
             - resource:
                 name: cpu
                 target:
                   averageUtilization: 75
                   type: Utilization
               type: Resource
             minReplicas: 3
           replicasRecommendation:
             replicas: 3
kind: List
metadata:
   resourceVersion: ""
   selfLink: ""
```

## Batch recommendation

Use a sample to show how to recommend all Deployments and StatefulSets by one `Analytics`:

```yaml
apiVersion: analysis.crane.io/v1alpha1
kind: Analytics
metadata:
   name: workload-replicas
   namespace: crane-system               # The Analytics in Crane-system will select all resource across all namespaces.
spec:
   type: Replicas                        # This can only be "Resource" or "Replicas".
   completionStrategy:
      completionStrategyType: Periodical  # This can only be "Once" or "Periodical".
      periodSeconds: 86400                # analytics selected resources every 1 day
   resourceSelectors:                    # defines all the resources to be select with
      - kind: Deployment
        apiVersion: apps/v1
      - kind: StatefulSet
        apiVersion: apps/v1
```

1. when using `crane-system` as your namespace，`Analytics` selected all namespaces，when namespace not equal `crane-system`，`Analytics` selected the resource that in `Analytics` namespace
2. resourceSelectors defines the resource to analysis，kind and apiVersion is mandatory，name is optional
3. resourceSelectors supoort any resource that are [Scale Subresource](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#scale-subresource)

## HPA Recommendation Algorithm model

### Inspecting

1. Workload with low replicas: If the replicas is too low,  may not be suitable for hpa recommendation. Associated configuration: `ehpa.deployment-min-replicas` | `ehpa.statefulset-min-replicas` | `ehpa.workload-min-replicas`
2. Workload with a certain percentage of not running pods: if the workload of Pod mostly can't run normally, may not be suitable for flexibility. Associated configuration: `ehpa.pod-min-ready-seconds` | `ehpa.pod-available-ratio`
3. Workload with low CPU usage: The low CPU usage workload means that there is no load pressure. In this case, we can't estimate it. Associated configuration: `ehpa.min-cpu-usage-threshold`
4. Workload with low fluctuation of CPU usage: dividing of the maximum and minimum usage is defined as the fluctuation rate. If the fluctuation rate is too low, the workload will not benefit much from hpa. Associated configuration: `ehpa.fluctuation-threshold` 

### Advising

In the advising phase, one EffectiveHPA Spec is recommended using the following Algorithm model. The recommended logic for each field is as follows:

**Recommend TargetUtilization**

Principle: Use Pod P99 resource utilization to recommend hpa. Because if the application can accept this utilization over P99 time, it can be inferred as a target for elasticity.

1. Get the Pod P99 usage of the past seven days by Percentile algorithm: $pod\_cpu\_usage\_p99$
2. Corresponding utilization:

      $target\_pod\_CPU\_utilization = \frac{pod\_cpu\_usage\_p99}{pod\_cpu\_request}$

3. To prevent over-utilization or under-utilization, target_pod_cpu_utilization needs to be less than ehpa.min-cpu-target-utilization and greater than ehpa. max-cpu-target-utilization

   $ehpa.max\mbox{-}cpu\mbox{-}target\mbox{-}utilization  < target\_pod\_cpu\_utilization < ehpa.min\mbox{-}cpu\mbox{-}target\mbox{-}utilization$

**Recommend minReplicas**

Principle: MinReplicas are recommended for the lowest hourly workload utilization for the past seven days.

1. Calculate the lowest median workload cpu usage of the past seven days: $workload\_cpu\_usage\_medium\_min$
2. Corresponding replicas: 

      $minReplicas = \frac{\mathrm{workload\_cpu\_usage\_medium\_min} }{pod\_cpu\_request \times ehpa.max-cpu-target-utilization}$

3. To prevent the minReplicas being too small, the minReplicas must be greater than or equal to ehpa.default-min-replicas

      $minReplicas \geq ehpa.default\mbox{-}min\mbox{-}replicas$

**Recommend maxReplicas**

Principle: Use workload's past and future seven days load to recommend maximum replicas.

1. Calculate P95 workload CPU usage for the past seven days and the next seven days: $workload\_cpu\_usage\_p95$
2. Corresponding replicas:

     $max\_replicas\_origin = \frac{\mathrm{workload\_cpu\_usage\_p95} }{pod\_cpu\_request \times target\_cpu\_utilization}$

3. To handle with the peak traffic, Magnify by a certain factor: 

   $max\_replicas = max\_replicas\_origin \times  ehpa.max\mbox{-}replicas\mbox{-}factor$

**Recommend MetricSpec(except CpuUtilization)**

1. If HPA is configured for workload, MetricSpecs other than CpuUtilization are inherited

**Recommend Behavior**

1. If HPA is configured for workload, the corresponding Behavior configuration is inherited

**Recommend Prediction**

1. Try to predict the CPU usage of the workload in the next seven days using DSP
2. If the prediction is successful, add the prediction configuration
3. If the workload is not predictable, do not add the prediction configuration.

## Configurations for HPA Recommendation

| Configuration | Default Value | Description |
| ------------- | ------------- | ----------- |
| ehpa.deployment-min-replicas | 1 | hpa recommendations are not made for workloads smaller than this value. |
| ehpa.statefulset-min-replicas| 1 | hpa recommendations are not made for workloads smaller than this value. |
| ehpa.workload-min-replicas| 1 | Workload replicas smaller than this value are not recommended for hpa. |
| ehpa.pod-min-ready-seconds| 30 | specifies the number of seconds in decide whether a POD is ready. |
| ehpa.pod-available-ratio| 0.5 | Workloads whose Ready pod ratio is smaller than this value are not recommended for hpa. |
| ehpa.default-min-replicas| 2 | the default minimum minReplicas.|
| ehpa.max-replicas-factor| 3 | the factor for calculate maxReplicas. |
| ehpa.min-cpu-usage-threshold| 10| hpa recommendations are not made for workloads smaller than this value.|
| ehpa.fluctuation-threshold| 1.5 | hpa recommendations are not made for workloads smaller than this value.|
| ehpa.min-cpu-target-utilization| 30 | |
| ehpa.max-cpu-target-utilization| 75 | |
| ehpa.reference-hpa| true | inherits the existing HPA configuration |
