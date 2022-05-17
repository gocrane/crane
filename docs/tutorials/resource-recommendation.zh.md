# 资源推荐

通过资源推荐，你可以得到集群中资源的推荐值并使用它提升集群的资源利用率。

## 创建资源分析

创建一个**资源分析** `Analytics`，这里我们通过实例 deployment: `nginx` 作为一个例子


=== "Main"

      ```bash
      kubectl apply -f https://raw.githubusercontent.com/gocrane/crane/main/examples/analytics/nginx-deployment.yaml
      kubectl apply -f https://raw.githubusercontent.com/gocrane/crane/main/examples/analytics/analytics-resource.yaml
      kubectl get analytics
      ```

=== "Mirror"

      ```bash
      kubectl apply -f https://finops.coding.net/p/gocrane/d/crane/git/raw/main/examples/analytics/nginx-deployment.yaml?download=false
      kubectl apply -f https://finops.coding.net/p/gocrane/d/crane/git/raw/main/examples/analytics/analytics-resource.yaml?download=false
      kubectl get analytics
      ```


```yaml title="analytics-resource.yaml"
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

结果如下:

```bash
NAME        AGE
nginx-resource   16m
```

查看 Analytics 的 Status，通过 status.recommendations[0].name 得到 Recommendation 的 name:

```bash
kubectl get analytics nginx-resource -o yaml
```

结果如下:

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

## 查看分析结果

查看 **Recommendation** 结果：

```bash
kubectl get recommend nginx-resource-resource-w45nq -o yaml
```

分析结果如下：

```yaml
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