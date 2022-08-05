# Resource Recommendation

Resource recommendation allows you to obtain recommended values for resources in a cluster and use them to improve the resource utilization of the cluster.

## Difference between VPA

Resource recommendations are a lightweight implementation of VPA and are more flexible.

1. Algorithm: The algorithm model adopts the Moving Window algorithm of VPA, and supports to customization algo args , providing higher flexibility
2. Support batch analysis: With the ResourceSelector, users can batch analyze multiple workloads without creating VPA objects one by one
3. More portable: It is difficult to use VPA's Auto mode in production because it will cause container reconstruction when updating container resource configuration. Resource recommendation provides suggestions to users and leaves the decision of change to users

## Create Resource Analytics

Create an **Resource** `Analytics` to give recommendation for deployment: `nginx-deployment` as a sample.


=== "Main"

      ```bash
      kubectl apply -f https://raw.githubusercontent.com/gocrane/crane/main/examples/analytics/nginx-deployment.yaml
      kubectl apply -f https://raw.githubusercontent.com/gocrane/crane/main/examples/analytics/analytics-resource.yaml
      kubectl get analytics

=== "Mirror"

      ```bash
      kubectl apply -f https://gitee.com/finops/crane/raw/main/examples/analytics/nginx-deployment.yaml
      kubectl apply -f https://gitee.com/finops/crane/raw/main/examples/analytics/analytics-resource.yaml
      kubectl get analytics
      ```

The created `Analytics` yaml is following:

```yaml title="analytics-resource.yaml"  hl_lines="7 24 11-14 28-31"
apiVersion: analysis.crane.io/v1alpha1
kind: Analytics
metadata:
  name: nginx-resource
spec:
  type: Resource                        # This can only be "Resource" or "Replicas".
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

You can get view analytics status by running:

```bash
kubectl get analytics nginx-resource -o yaml
```

The output is similar to:

```yaml
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

## Recommendation: Analytics result

You can get recommendations that created by above `Analytics` by running.

```bash
kubectl get recommend -l analysis.crane.io/analytics-name=nginx-resource -o yaml
```

The output is similar to:

```yaml 
apiVersion: v1
items:
- apiVersion: analysis.crane.io/v1alpha1
  kind: Recommendation
  metadata:
    creationTimestamp: "2022-06-15T15:26:25Z"
    generateName: nginx-resource-resource-
    generation: 1
    labels:
      analysis.crane.io/analytics-name: nginx-resource
      analysis.crane.io/analytics-type: Resource
      analysis.crane.io/analytics-uid: 9e78964b-f8ae-40de-9740-f9a715d16280
      app: nginx
    name: nginx-resource-resource-t4xpn
    namespace: default
    ownerReferences:
    - apiVersion: analysis.crane.io/v1alpha1
      blockOwnerDeletion: false
      controller: false
      kind: Analytics
      name: nginx-resource
      uid: 9e78964b-f8ae-40de-9740-f9a715d16280
    resourceVersion: "2117439429"
    selfLink: /apis/analysis.crane.io/v1alpha1/namespaces/default/recommendations/nginx-resource-resource-t4xpn
    uid: 8005e3e0-8fe9-470b-99cf-5ce9dd407529
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
      resourceRequest:
        containers:
        - containerName: nginx
          target:
            cpu: 100m
            memory: 100Mi
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""
```

The `status.recommendedValue.ResourceRequest` is recommended by crane's recommendation engine.

## Batch recommendation

Use a sample to show how to recommend all Deployments and StatefulSets by one `Analytics`:

```yaml
apiVersion: analysis.crane.io/v1alpha1
kind: Analytics
metadata:
  name: workload-resource
  namespace: crane-system               # The Analytics in Crane-system will select all resource across all namespaces.
spec:
  type: Resource                        # This can only be "Resource" or "Replicas".
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


## Resource Recommendation Algorithm model

### Inspecting

Workload with not pods: if the workload has no pods exist means that it's not a available workload.

### Advising

VPA's Moving Window algorithm was used to calculate the CPU and Memory of each container and give the corresponding recommended values
