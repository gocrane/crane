# 副本數推薦

Kubernetes 用戶在創建應用資源時常常是基於經驗值來設置副本數或者 EHPA 配置。通過副本數推薦的算法分析應用的真實用量推薦更合適的副本配置，您可以參考並採納它提升集群的資源利用率。

## 產品功能

1. 算法：計算副本數的算法參考了 HPA 的計算公式，並且支持自定義算法的關鍵配置
2. HPA 推薦：副本數推薦會掃描出適合配置水平彈性（EHPA）的應用，並給出 EHPA 的配置, [EHPA](using-effective-hpa-to-scaling-with-effectiveness.md) 是 Crane 提供了智能水平彈性產品
3. 支持批量分析：通過 `Analytics` 的 ResourceSelector，用戶可以批量分析多個工作負載

## 創建彈性分析

創建一個**彈性分析** `Analytics`，這裡我們通過實例 deployment: `nginx` 作為一個例子

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

結果如下:

```bash
NAME             AGE
nginx-replicas   16m
```

查看 Analytics 詳情:

```bash
kubectl get analytics nginx-replicas -o yaml
```

結果如下:

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

## 查看分析結果

查看 **Recommendation** 結果：

```bash
kubectl get recommend -l analysis.crane.io/analytics-name=nginx-replicas -o yaml
```

分析結果如下：
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

## 批量推薦

我們通過一個例子來演示如何使用 `Analytics` 推薦集群中所有的 Deployment 和 StatefulSet：

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

1. 當 namespace 等於 `crane-system` 時，`Analytics` 選擇的資源是集群中所有的 namespace，當 namespace 不等於 `crane-system` 時，`Analytics` 選擇 `Analytics` namespace 下的資源
2. resourceSelectors 通過數組配置需要分析的資源，kind 和 apiVersion 是必填字段，name 選填
3. resourceSelectors 支持配置任意支持 [Scale Subresource](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#scale-subresource) 的資源

## 彈性推薦計算模型

### 篩選階段

1. 低副本數的工作負載: 過低的副本數可能彈性需求不高，關聯配置: `ehpa.deployment-min-replicas` | `ehpa.statefulset-min-replicas` | `ehpa.workload-min-replicas`
2. 存在一定比例非 Running Pod 的工作負載: 如果工作負載的 Pod 大多不能正常運行，可能不適合彈性，關聯配置: `ehpa.pod-min-ready-seconds` | `ehpa.pod-available-ratio`
3. 低 CPU 使用量的工作負載: 過低使用量的工作負載意味著沒有業務壓力，此時通過使用率推薦彈性不准，關聯配置: `ehpa.min-cpu-usage-threshold`
4. CPU 使用量的波動率過低: 使用量的最大值和最小值的倍數定義為波動率，波動率過低的工作負載通過彈性降本的收益不大，關聯配置: `ehpa.fluctuation-threshold`

### 推薦

推薦階段通過以下模型推荐一個 EffectiveHPA 的 Spec。每個字段的推薦邏輯如下：

**推薦 TargetUtilization**

原理: 使用 Pod P99 資源利用率推薦彈性的目標。因為如果應用可以在 P99 時間內接受這個利用率，可以推斷出可作為彈性的目標。

 1. 通過 Percentile 算法得到 Pod 過去七天 的 P99 使用量: $pod\_cpu\_usage\_p99$
 2. 對應的利用率:
 
      $target\_pod\_CPU\_utilization = \frac{pod\_cpu\_usage\_p99}{pod\_cpu\_request}$

 3. 為了防止利用率過大或過小，target_pod_cpu_utilization 需要小於 ehpa.min-cpu-target-utilization 和大於 ehpa.max-cpu-target-utilization

    $ehpa.max\mbox{-}cpu\mbox{-}target\mbox{-}utilization  < target\_pod\_cpu\_utilization < ehpa.min\mbox{-}cpu\mbox{-}target\mbox{-}utilization$

**推薦 minReplicas**

原理: 使用 workload 過去七天內每小時負載最低的利用率推薦 minReplicas。

1. 計算過去7天 workload 每小時使用量中位數的最低值: $workload\_cpu\_usage\_medium\_min$
2. 對應的最低利用率對應的副本數: 

     $minReplicas = \frac{\mathrm{workload\_cpu\_usage\_medium\_min} }{pod\_cpu\_request \times ehpa.max-cpu-target-utilization}$

3. 為了防止 minReplicas 過小，minReplicas 需要大於等於 ehpa.default-min-replicas

     $minReplicas \geq ehpa.default\mbox{-}min\mbox{-}replicas$

**推薦 maxReplicas**

原理: 使用 workload 過去和未來七天的負載推薦最大副本數。

1. 計算過去七天和未來七天 workload cpu 使用量的 P95: $workload\_cpu\_usage\_p95$
2. 對應的副本數:

     $max\_replicas\_origin = \frac{\mathrm{workload\_cpu\_usage\_p95} }{pod\_cpu\_request \times target\_cpu\_utilization}$

3. 為了應對流量洪峰，放大一定倍數:

     $max\_replicas = max\_replicas\_origin \times  ehpa.max\mbox{-}replicas\mbox{-}factor$

**推薦CPU以外 MetricSpec**

1. 如果 workload 配置了 HPA，繼承相應除 CpuUtilization 以外的其他 MetricSpec
**推薦 Behavior**

1. 如果 workload 配置了 HPA，繼承相應的 Behavior 配置

**預測**

1. 嘗試預測工作負載未來七天的 CPU 使用量，算法是 DSP
2. 如果預測成功則添加預測配置
3. 如果不可預測則不添加預測配置，退化成不具有預測功能的 EffectiveHPA

## 彈性分析計算配置

| 配置項 | 默認值 | 描述|
| ------------- | ------------- | ----------- |
| ehpa.deployment-min-replicas | 1 | 小於該值的工作負載不做彈性推薦 |
| ehpa.statefulset-min-replicas| 1 | 小於該值的工作負載不做彈性推薦 |
| ehpa.workload-min-replicas| 1 | 小於該值的工作負載不做彈性推薦 |
| ehpa.pod-min-ready-seconds| 30 | 定義了 Pod 是否 Ready 的秒數 |
| ehpa.pod-available-ratio| 0.5 | Ready Pod 比例小於該值的工作負載不做彈性推薦 |
| ehpa.default-min-replicas| 2 | 最小 minReplicas |
| ehpa.max-replicas-factor| 3 | 計算 maxReplicas 的倍數 |
| ehpa.min-cpu-usage-threshold| 10| 小於該值的工作負載不做彈性推薦 |
| ehpa.fluctuation-threshold| 1.5 | 小於該值的工作負載不做彈性推薦 |
| ehpa.min-cpu-target-utilization| 30 | |
| ehpa.max-cpu-target-utilization| 75 | |
| ehpa.reference-hpa| true | 繼承現有的 HPA 配置 |