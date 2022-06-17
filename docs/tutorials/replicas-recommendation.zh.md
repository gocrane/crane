# 弹性推荐

通过弹性推荐，你可以发现集群中适合弹性的资源，并使用 Crane 推荐的弹性配置创建自动弹性器: [Effective HorizontalPodAutoscaler](using-effective-hpa-to-scaling-with-effectiveness.md)

## 创建弹性分析

创建一个**弹性分析** `Analytics`，这里我们通过实例 deployment: `nginx` 作为一个例子

=== "Main"

      ```bash
      kubectl apply -f https://raw.githubusercontent.com/gocrane/crane/main/examples/analytics/nginx-deployment.yaml
      kubectl apply -f https://raw.githubusercontent.com/gocrane/crane/main/examples/analytics/analytics-hpa.yaml
      kubectl get analytics
      ```
 
=== "Mirror"

      ```bash
      kubectl apply -f https://finops.coding.net/p/gocrane/d/crane/git/raw/main/examples/analytics/nginx-deployment.yaml?download=false
      kubectl apply -f https://finops.coding.net/p/gocrane/d/crane/git/raw/main/examples/analytics/analytics-hpa.yaml?download=false
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

结果如下:

```bash
NAME        AGE
nginx-hpa   16m
```

查看 Analytics 的 Status，通过 status.recommendations[0].name 得到 Recommendation 的 name:

```bash
kubectl get analytics nginx-hpa -o yaml
```

结果如下:

```yaml hl_lines="32"
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

## 查看分析结果

查看 **Recommendation** 结果：

```bash
kubectl get recommend nginx-hpa-hpa-cd86s -o yaml
```

分析结果如下：

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
