# Resource Recommendation

Resource recommendation allows you to obtain recommended values for resources in a cluster and use them to improve the resource utilization of the cluster.

## Create Resource Analytics

Create an **Resource** `Analytics` to give recommendation for deployment: `nginx-deployment` as a sample.


=== "Main"

      ```bash
      kubectl apply -f https://raw.githubusercontent.com/gocrane/crane/main/examples/analytics/nginx-deployment.yaml
      kubectl apply -f https://raw.githubusercontent.com/gocrane/crane/main/examples/analytics/analytics-resource.yaml
      kubectl get analytics -n crane-system
      ```

=== "Mirror"

      ```bash
      kubectl apply -f https://finops.coding.net/p/gocrane/d/crane/git/raw/main/examples/analytics/nginx-deployment.yaml?download=false
      kubectl apply -f https://finops.coding.net/p/gocrane/d/crane/git/raw/main/examples/analytics/analytics-resource.yaml?download=false
      kubectl get analytics -n crane-system
      ```


```yaml title="analytics-resource.yaml"  hl_lines="7 24 11-14 28-31"
apiVersion: analysis.crane.io/v1alpha1
kind: Analytics
metadata:
  name: nginx-resource
spec:
  type: Resource                        # This can only be "Resource" or "HPA".
  completionStrategy:
    completionStrategyType: Periodical  # This can only be "Once" or "Periodical".
    periodSeconds: 86400                # analytics selected resources every 1 day
  resourceSelectors:                    # defines all the resources to be select with
    - kind: Deployment
      apiVersion: apps/v1
      name: nginx-deployment
```

The output is:

```bash
NAME             AGE
nginx-resource   16m
```

You can get created recommendation from analytics status:

```bash
kubectl get analytics nginx-resource -o yaml
```

The output is similar to:

```yaml hl_lines="27"
apiVersion: analysis.crane.io/v1alpha1
kind: Analytics
metadata:
  name: nginx-resource
  namespace: default
spec:
  completionStrategy:
    completionStrategyType: Periodical
    periodSeconds: 86400
  resourceSelectors:
    - apiVersion: apps/v1
      kind: Deployment
      labelSelector: {}
      name: nginx-deployment
  type: Resource
status:
  conditions:
    - lastTransitionTime: "2022-05-15T14:38:35Z"
      message: Analytics is ready
      reason: AnalyticsReady
      status: "True"
      type: Ready
  lastUpdateTime: "2022-05-15T14:38:35Z"
  recommendations:
    - lastStartTime: "2022-05-15T14:38:35Z"
      message: Success
      name: nginx-resource-resource-w45nq
      namespace: default
      targetRef:
        apiVersion: apps/v1
        kind: Deployment
        name: nginx-deployment
        namespace: default
      uid: 750cb3bd-0b87-4f87-acbe-57e621af0a1e 
```

The recommendation name presents on `status.recommendations[0].name`. Then you can get recommendation detail by running:

## Recommendation: Analytics result

```bash
kubectl get recommend -n crane-system nginx-resource-resource-w45nq -o yaml
```

The output is similar to:

```yaml  hl_lines="32-37"
apiVersion: analysis.crane.io/v1alpha1
kind: Recommendation
metadata:
  creationTimestamp: "2022-05-15T14:38:35Z"
  generateName: nginx-resource-resource-
  generation: 1
  labels:
    analysis.crane.io/analytics-name: nginx-resource
    analysis.crane.io/analytics-type: Resource
    analysis.crane.io/analytics-uid: 89e6d867-d639-4255-89cf-a3436dad6251
    app: nginx
  name: nginx-resource-resource-w45nq
  namespace: default
  ownerReferences:
    - apiVersion: analysis.crane.io/v1alpha1
      blockOwnerDeletion: false
      controller: false
      kind: Analytics
      name: nginx-resource
      uid: 89e6d867-d639-4255-89cf-a3436dad6251
  resourceVersion: "541878166"
  selfLink: /apis/analysis.crane.io/v1alpha1/namespaces/default/recommendations/nginx-resource-resource-w45nq
  uid: 750cb3bd-0b87-4f87-acbe-57e621af0a1e
spec:
  adoptionType: StatusAndAnnotation
  completionStrategy:
    completionStrategyType: Once
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: nginx-deployment
    namespace: default
  type: Resource
status:
  recommendedValue: |
    containers:
    - containerName: nginx
      target:
        cpu: 114m
        memory: "120586239"
```

The `status.resourceRequest` is recommended by crane's recommendation engine.

Something you should know about Resource recommendation:

* Resource Recommendation use historic prometheus metrics to calculate and propose.
* We use **Percentile** algorithm to process metrics that also used by VPA.
* If the workload is running for a long term like several weeks, the result will be more accurate.
