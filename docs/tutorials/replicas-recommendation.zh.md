# 副本数推荐

Kubernetes 用户在创建应用资源时常常是基于经验值来设置副本数或者 EHPA 配置。通过副本数推荐的算法分析应用的真实用量推荐更合适的副本配置，您可以参考并采纳它提升集群的资源利用率。

## 产品功能

1. 算法：计算副本数的算法参考了 HPA 的计算公式，并且支持自定义算法的关键配置
2. HPA 推荐：副本数推荐会扫描出适合配置水平弹性（EHPA）的应用，并给出 EHPA 的配置, [EHPA](using-effective-hpa-to-scaling-with-effectiveness.md) 是 Crane 提供了智能水平弹性产品
3. 支持批量分析：通过 `Analytics` 的 ResourceSelector，用户可以批量分析多个工作负载

## 创建弹性分析

创建一个**弹性分析** `Analytics`，这里我们通过实例 deployment: `nginx` 作为一个例子

=== "Main"

      ```bash
      kubectl apply -f https://raw.githubusercontent.com/gocrane/crane/main/examples/analytics/nginx-deployment.yaml
      kubectl apply -f https://raw.githubusercontent.com/gocrane/crane/main/examples/analytics/analytics-replicas.yaml
      kubectl get analytics
      ```
 
=== "Mirror"

      ```bash
      kubectl apply -f https://finops.coding.net/p/gocrane/d/crane/git/raw/main/examples/analytics/nginx-deployment.yaml?download=false
      kubectl apply -f https://finops.coding.net/p/gocrane/d/crane/git/raw/main/examples/analytics/analytics-replicas.yaml?download=false
      kubectl get analytics
      ```


```yaml title="analytics-replicas.yaml"
apiVersion: analysis.crane.io/v1alpha1
kind: Analytics
metadata:
  name: nginx-replicas
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
    replicas.workload-min-replicas: "1"
    replicas.fluctuation-threshold: "0"
    replicas.min-cpu-usage-threshold: "0"
```

结果如下:

```bash
NAME             AGE
nginx-replicas   16m
```

查看 Analytics 详情:

```bash
kubectl get analytics nginx-replicas -o yaml
```

结果如下:

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
  - lastTransitionTime: "2022-06-17T06:56:07Z"
    message: Analytics is ready
    reason: AnalyticsReady
    status: "True"
    type: Ready
  lastUpdateTime: "2022-06-17T06:56:06Z"
  recommendations:
  - lastStartTime: "2022-06-17T06:56:06Z"
    message: Success
    name: nginx-replicas-replicas-wq6wm
    namespace: default
    targetRef:
      apiVersion: apps/v1
      kind: Deployment
      name: nginx-deployment
      namespace: default
    uid: 59f3eb3c-f786-4b15-b37e-774e5784c2db
```

## 查看分析结果

查看 **Recommendation** 结果：

```bash
kubectl get recommend -l analysis.crane.io/analytics-name=nginx-replicas -o yaml
```

分析结果如下：

```yaml
apiVersion: v1
items:
  - apiVersion: analysis.crane.io/v1alpha1
    kind: Recommendation
    metadata:
      creationTimestamp: "2022-06-17T06:56:06Z"
      generateName: nginx-replicas-replicas-
      generation: 2
      labels:
        analysis.crane.io/analytics-name: nginx-replicas
        analysis.crane.io/analytics-type: Replicas
        analysis.crane.io/analytics-uid: 795f245b-1e1f-4f7b-a02b-885d7a495e5b
        app: nginx
      name: nginx-replicas-replicas-wq6wm
      namespace: default
      ownerReferences:
        - apiVersion: analysis.crane.io/v1alpha1
          blockOwnerDeletion: false
          controller: false
          kind: Analytics
          name: nginx-replicas
          uid: 795f245b-1e1f-4f7b-a02b-885d7a495e5b
      resourceVersion: "2182455668"
      selfLink: /apis/analysis.crane.io/v1alpha1/namespaces/default/recommendations/nginx-replicas-replicas-wq6wm
      uid: 59f3eb3c-f786-4b15-b37e-774e5784c2db
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
        - lastTransitionTime: "2022-06-17T06:56:07Z"
          message: Recommendation is ready
          reason: RecommendationReady
          status: "True"
          type: Ready
      lastUpdateTime: "2022-06-17T06:56:07Z"
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

## 批量推荐

我们通过一个例子来演示如何使用 `Analytics` 推荐集群中所有的 Deployment 和 StatefulSet：

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

1. 当 namespace 等于 `crane-system` 时，`Analytics` 选择的资源是集群中所有的 namespace，当 namespace 不等于 `crane-system` 时，`Analytics` 选择 `Analytics` namespace 下的资源
2. resourceSelectors 通过数组配置需要分析的资源，kind 和 apiVersion 是必填字段，name 选填
3. resourceSelectors 支持配置任意支持 [Scale Subresource](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#scale-subresource) 的资源

## 弹性推荐计算模型

### 筛选阶段

1. 低副本数的工作负载: 过低的副本数可能弹性需求不高，关联配置: `ehpa.deployment-min-replicas` | `ehpa.statefulset-min-replicas` | `ehpa.workload-min-replicas`
2. 存在一定比例非 Running Pod 的工作负载: 如果工作负载的 Pod 大多不能正常运行，可能不适合弹性，关联配置: `ehpa.pod-min-ready-seconds` | `ehpa.pod-available-ratio`
3. 低 CPU 使用量的工作负载: 过低使用量的工作负载意味着没有业务压力，此时通过使用率推荐弹性不准，关联配置: `ehpa.min-cpu-usage-threshold`
4. CPU 使用量的波动率过低: 使用量的最大值和最小值的倍数定义为波动率，波动率过低的工作负载通过弹性降本的收益不大，关联配置: `ehpa.fluctuation-threshold`

### 推荐

推荐阶段通过以下模型推荐一个 EffectiveHPA 的 Spec。每个字段的推荐逻辑如下：

**推荐 TargetUtilization**

原理: 使用 Pod P99 资源利用率推荐弹性的目标。因为如果应用可以在 P99 时间内接受这个利用率，可以推断出可作为弹性的目标。

 1. 通过 Percentile 算法得到 Pod 过去七天 的 P99 使用量: $pod\_cpu\_usage\_p99$
 2. 对应的利用率:
 
      $target\_pod\_CPU\_utilization = \frac{pod\_cpu\_usage\_p99}{pod\_cpu\_request}$

 3. 为了防止利用率过大或过小，target_pod_cpu_utilization 需要小于 ehpa.min-cpu-target-utilization 和大于 ehpa.max-cpu-target-utilization

    $ehpa.max\mbox{-}cpu\mbox{-}target\mbox{-}utilization  < target\_pod\_cpu\_utilization < ehpa.min\mbox{-}cpu\mbox{-}target\mbox{-}utilization$

**推荐 minReplicas**

原理: 使用 workload 过去七天内每小时负载最低的利用率推荐 minReplicas。

1. 计算过去7天 workload 每小时使用量中位数的最低值: $workload\_cpu\_usage\_medium\_min$
2. 对应的最低利用率对应的副本数: 

     $minReplicas = \frac{\mathrm{workload\_cpu\_usage\_medium\_min} }{pod\_cpu\_request \times ehpa.max-cpu-target-utilization}$

3. 为了防止 minReplicas 过小，minReplicas 需要大于等于 ehpa.default-min-replicas

     $minReplicas \geq ehpa.default\mbox{-}min\mbox{-}replicas$

**推荐 maxReplicas**

原理: 使用 workload 过去和未来七天的负载推荐最大副本数。

1. 计算过去七天和未来七天 workload cpu 使用量的 P95: $workload\_cpu\_usage\_p95$
2. 对应的副本数:

     $max\_replicas\_origin = \frac{\mathrm{workload\_cpu\_usage\_p95} }{pod\_cpu\_request \times target\_cpu\_utilization}$

3. 为了应对流量洪峰，放大一定倍数:

     $max\_replicas = max\_replicas\_origin \times  ehpa.max\mbox{-}replicas\mbox{-}factor$

**推荐CPU以外 MetricSpec**

1. 如果 workload 配置了 HPA，继承相应除 CpuUtilization 以外的其他 MetricSpec

**推荐 Behavior**

1. 如果 workload 配置了 HPA，继承相应的 Behavior 配置

**预测**

1. 尝试预测工作负载未来七天的 CPU 使用量，算法是 DSP
2. 如果预测成功则添加预测配置
3. 如果不可预测则不添加预测配置，退化成不具有预测功能的 EffectiveHPA

## 弹性分析计算配置

| 配置项 | 默认值 | 描述|
| ------------- | ------------- | ----------- |
| ehpa.deployment-min-replicas | 1 | 小于该值的工作负载不做弹性推荐 |
| ehpa.statefulset-min-replicas| 1 | 小于该值的工作负载不做弹性推荐 |
| ehpa.workload-min-replicas| 1 | 小于该值的工作负载不做弹性推荐 |
| ehpa.pod-min-ready-seconds| 30 | 定义了 Pod 是否 Ready 的秒数 |
| ehpa.pod-available-ratio| 0.5 | Ready Pod 比例小于该值的工作负载不做弹性推荐 |
| ehpa.default-min-replicas| 2 | 最小 minReplicas |
| ehpa.max-replicas-factor| 3 | 计算 maxReplicas 的倍数 |
| ehpa.min-cpu-usage-threshold| 10| 小于该值的工作负载不做弹性推荐 |
| ehpa.fluctuation-threshold| 1.5 | 小于该值的工作负载不做弹性推荐 |
| ehpa.min-cpu-target-utilization| 30 | |
| ehpa.max-cpu-target-utilization| 75 | |
| ehpa.reference-hpa| true | 继承现有的 HPA 配置 |


