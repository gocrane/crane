# Analytics and Recommendation

Analytics and Recommendation provide capacity that analyzes the workload in k8s cluster and provide recommendations about resource optimize.

Two Recommendations are currently supported:

- **ResourceRecommend**: Recommend container requests & limit resources based on historic metrics.
- **Effective HPARecommend**: Recommend which workloads are suitable for autoscaling and provide optimized configurations such as minReplicas, maxReplicas.

## Analytics and Recommend Pod Resources

Create an **Resource** `Analytics` to give recommendation for deployment: `craned` and `metric-adapter` as a sample.

```bash
kubectl apply -f https://raw.githubusercontent.com/gocrane/crane/main/examples/analytics/analytics-resource.yaml
kubectl get analytics -n crane-system
```

```yaml title="analytics-resource.yaml"  hl_lines="7 24 11-14 28-31"
apiVersion: analysis.crane.io/v1alpha1
kind: Analytics
metadata:
  name: craned-resource
  namespace: crane-system
spec:
  type: Resource                        # This can only be "Resource" or "HPA".
  completionStrategy:
    completionStrategyType: Periodical  # This can only be "Once" or "Periodical".
    periodSeconds: 86400                # analytics selected resources every 1 day
  resourceSelectors:                    # defines all the resources to be select with
    - kind: Deployment
      apiVersion: apps/v1
      name: craned

---

apiVersion: analysis.crane.io/v1alpha1
kind: Analytics
metadata:
  name: metric-adapter-resource
  namespace: crane-system
spec:
  type: Resource                       # This can only be "Resource" or "HPA".
  completionStrategy:
    completionStrategyType: Periodical # This can only be "Once" or "Periodical".
    periodSeconds: 3600                # analytics selected resources every 1 hour
  resourceSelectors:                   # defines all the resources to be select with
    - kind: Deployment
      apiVersion: apps/v1
      name: metric-adapter
```

The output is:

```bash
NAME                      AGE
craned-resource           15m
metric-adapter-resource   15m
```

You can get created recommendation from analytics status:

```bash
kubectl get analytics craned-resource -n crane-system -o yaml
```

The output is similar to:

```yaml hl_lines="18-21"
apiVersion: analysis.crane.io/v1alpha1
kind: Analytics
metadata:
  name: craned-resource
  namespace: crane-system
spec:
  completionStrategy:
    completionStrategyType: Periodical
    periodSeconds: 86400
  resourceSelectors:
  - apiVersion: apps/v1
    kind: Deployment
    labelSelector: {}
    name: craned
  type: Resource
status:
  lastSuccessfulTime: "2022-01-12T08:40:59Z"
  recommendations:
  - name: craned-resource-resource-j7shb
    namespace: crane-system
    uid: 8ce2eedc-7969-4b80-8aee-fd4a98d6a8b6    
```

The recommendation name presents on `status.recommendations[0].name`. Then you can get recommendation detail by running:

```bash
kubectl get recommend -n crane-system craned-resource-resource-j7shb -o yaml
```

The output is similar to:

```yaml  hl_lines="32-37"
apiVersion: analysis.crane.io/v1alpha1
kind: Recommendation
metadata:
  name: craned-resource-resource-j7shb
  namespace: crane-system
  ownerReferences:
  - apiVersion: analysis.crane.io/v1alpha1
    blockOwnerDeletion: false
    controller: false
    kind: Analytics
    name: craned-resource
    uid: a9e6dc0d-ab26-4f2a-84bd-4fe9e0f3e105
spec:
  completionStrategy:
    completionStrategyType: Periodical
    periodSeconds: 86400
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: craned
    namespace: crane-system
  type: Resource
status:
  conditions:
  - lastTransitionTime: "2022-01-12T08:40:59Z"
    message: Recommendation is ready
    reason: RecommendationReady
    status: "True"
    type: Ready
  lastSuccessfulTime: "2022-01-12T08:40:59Z"
  lastUpdateTime: "2022-01-12T08:40:59Z"
  resourceRequest:
    containers:
    - containerName: craned
      target:
        cpu: 114m
        memory: 120586239m
```

The `status.resourceRequest` is recommended by crane's recommendation engine.

Something you should know about Resource recommendation:

* Resource Recommendation use historic prometheus metrics to calculate and propose.
* We use **Percentile** algorithm to process metrics that also used by VPA.
* If the workload is running for a long term like several weeks, the result will be more accurate.

## Analytics and Recommend HPA

Create an **HPA** `Analytics` to give recommendations for deployment: `craned` and `metric-adapter` as a sample.

```bash
kubectl apply -f https://raw.githubusercontent.com/gocrane/crane/main/examples/analytics/analytics-hpa.yaml
kubectl get analytics -n crane-system 
```

```yaml title="analytics-hpa.yaml" hl_lines="7 24 11-14 28-31"
apiVersion: analysis.crane.io/v1alpha1
kind: Analytics
metadata:
  name: craned-hpa
  namespace: crane-system
spec:
  type: HPA                        # This can only be "Resource" or "HPA".
  completionStrategy:
    completionStrategyType: Periodical  # This can only be "Once" or "Periodical".
    periodSeconds: 600                  # analytics selected resources every 10 minutes
  resourceSelectors:                    # defines all the resources to be select with
    - kind: Deployment
      apiVersion: apps/v1
      name: craned

---

apiVersion: analysis.crane.io/v1alpha1
kind: Analytics
metadata:
  name: metric-adapter-hpa
  namespace: crane-system
spec:
  type: HPA                       # This can only be "Resource" or "HPA".
  completionStrategy:
    completionStrategyType: Periodical # This can only be "Once" or "Periodical".
    periodSeconds: 3600                # analytics selected resources every 1 hour
  resourceSelectors:                   # defines all the resources to be select with
    - kind: Deployment
      apiVersion: apps/v1
      name: metric-adapter
```


The output is:

```bash
NAME                      AGE
craned-hpa                5m52s
craned-resource           18h
metric-adapter-hpa        5m52s
metric-adapter-resource   18h

```

You can get created recommendation from analytics status:

```bash
kubectl get analytics craned-hpa -n crane-system -o yaml
```

The output is similar to:

```yaml hl_lines="21"
apiVersion: analysis.crane.io/v1alpha1
kind: Analytics
metadata:
  name: craned-hpa
  namespace: crane-system
spec:
  completionStrategy:
    completionStrategyType: Periodical
    periodSeconds: 86400
  resourceSelectors:
  - apiVersion: apps/v1
    kind: Deployment
    labelSelector: {}
    name: craned
  type: HPA
status:
  lastSuccessfulTime: "2022-01-13T07:26:18Z"
  recommendations:
  - apiVersion: analysis.crane.io/v1alpha1
    kind: Recommendation
    name: craned-hpa-hpa-2f22w
    namespace: crane-system
    uid: 397733ee-986a-4630-af75-736d2b58bfac
```

The recommendation name presents on `status.recommendations[0].name`. Then you can get recommendation detail by running:

```bash
kubectl get recommend -n crane-system craned-resource-resource-2f22w -o yaml
```

The output is similar to:

```yaml hl_lines="26-29"
apiVersion: analysis.crane.io/v1alpha1
kind: Recommendation
metadata:
  name: craned-hpa-hpa-2f22w
  namespace: crane-system
  ownerReferences:
  - apiVersion: analysis.crane.io/v1alpha1
    blockOwnerDeletion: false
    controller: false
    kind: Analytics
    name: craned-hpa
    uid: b216d9c3-c52e-4c9c-b9e9-9d5b45165b1d
spec:
  completionStrategy:
    completionStrategyType: Periodical
    periodSeconds: 86400
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: craned
    namespace: crane-system
  type: HPA
status:
  conditions:
  - lastTransitionTime: "2022-01-13T07:51:18Z"
    message: 'Failed to offer recommend, Recommendation crane-system/craned-hpa-hpa-2f22w
      error EHPAAdvisor prediction metrics data is unexpected, List length is 0 '
    reason: FailedOfferRecommend
    status: "False"
    type: Ready
  lastUpdateTime: "2022-01-13T07:51:18Z"
```

The `status.resourceRequest` is recommended by crane's recommendation engine. The fail reason is demo workload don't have enough run time.

Something you should know about HPA recommendation:

* HPA Recommendation use historic prometheus metrics to calculate, forecast and propose.
* We use **DSP** algorithm to process metrics.
* We recommend using Effective HorizontalPodAutoscaler to execute autoscaling, you can see [this document](using-time-series-prediction.md) to learn more.
* The Workload need match following conditions:
    * Existing at least one ready pod
    * Ready pod ratio should larger that 50%
    * Must provide cpu request for pod spec
    * The workload should be running for at least **a week** to get enough metrics to forecast
    * The workload's cpu load should be predictable, **too low** or **too unstable** workload often is unpredictable
