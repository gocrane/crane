# 资源推荐

Kubernetes 用户在创建应用资源时常常是基于经验值来设置 request 和 limit。通过资源推荐的算法分析应用的真实用量推荐更合适的资源配置，您可以参考并采纳它提升集群的资源利用率。

## 产品功能

资源推荐是 VPA 的轻量化实现，且更灵活。

1. 算法：算法模型采用了 VPA 的滑动窗口（Moving Window）算法，并且支持自定义算法的关键配置，提供了更高的灵活性
2. 支持批量分析：通过 `Analytics` 的 ResourceSelector，用户可以批量分析多个工作负载，而无需一个一个的创建 VPA 对象
3. 更轻便：由于 VPA 的 Auto 模式在更新容器资源配置时会导致容器重建，因此很难在生产上使用自动模式，资源推荐给用户提供资源建议，把变更的决定交给用户决定

## 创建资源分析

我们通过 deployment: `nginx` 和 `Analytics` 作为一个例子演示如何开始一次资源推荐之旅：


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

查看 Analytics 详情:

```bash
kubectl get analytics nginx-resource -o yaml
```

结果如下:

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

## 查看分析结果

查看分析结果 **Recommendation**：

```bash
kubectl get recommend -l analysis.crane.io/analytics-name=nginx-resource -o yaml
```

分析结果如下：

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

## 批量推荐

我们通过一个例子来演示如何使用 `Analytics` 推荐集群中所有的 Deployment 和 StatefulSet：

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

1. 当 namespace 等于 `crane-system` 时，`Analytics` 选择的资源是集群中所有的 namespace，当 namespace 不等于 `crane-system` 时，`Analytics` 选择 `Analytics` namespace 下的资源
2. resourceSelectors 通过数组配置需要分析的资源，kind 和 apiVersion 是必填字段，name 选填
3. resourceSelectors 支持配置任意支持 [Scale Subresource](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#scale-subresource) 的资源

## 资源推荐计算模型

### 筛选阶段

没有 Pod 的工作负载: 如果工作负载没有 Pod，无法进行算法分析。

### 推荐

采用 VPA 的滑动窗口（Moving Window）算法分别计算每个容器的 CPU 和 Memory 并给出对应的推荐值

## 常见问题

### 如何让推荐结果更准确

应用在监控系统（比如 Prometheus）中的历史数据越久，推荐结果就越准确，建议生产上超过两周时间。对新建应用的预测往往不准，可以通过参数配置保证只对历史数据长度超过一定天数的业务推荐。




