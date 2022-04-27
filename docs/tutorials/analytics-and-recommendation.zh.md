

# 资源的分析和推荐

资源的分析和建议模块，用于分析 Kuberentes 群集中的工作负载，并提供资源优化建议值，即为工作负载推荐更合理的 Request 数值。

目前支持两种推荐：

- **资源推荐**：根据容器历史资源用量推荐容器的 Request 和 Limit。
- **Effective HPA 推荐**：建议哪些工作负载适合自动缩放，并提供优化的配置选项，例如 minReplicas、maxReplicas。

## 资源推荐使用方式

1. [部署 Crane](https://docs.gocrane.io/dev/zh/installation/)，当相关的资源对象成功运行时，表示 Crane 的组件已部署完成。如下所示：

   ```shell
   NAME                            READY   UP-TO-DATE   AVAILABLE   AGE
   craned                          1/1     1            1           31m
   fadvisor                        1/1     1            1           41m
   grafana                         1/1     1            1           42m
   metric-adapter                  1/1     1            1           31m
   prometheus-kube-state-metrics   1/1     1            1           43m
   prometheus-server               1/1     1            1           43m
   ```

2. 在集群里部署“Analytics”类型资源对象，作用在名为 `craned` and `metric-adapter` 的 Deployment 上：

```bash
kubectl apply -f https://raw.githubusercontent.com/gocrane/crane/main/examples/analytics/analytics-resource.yaml
```

```yaml
# 上述命令操作的具体 YAML 如下：
apiVersion: analysis.crane.io/v1alpha1
kind: Analytics
metadata:
  name: craned-resource
  namespace: crane-system
spec:
  type: Resource                        # 只能是 "Resource" 或 "HPA"，Resource 表示资源推荐，HPA 表示 Effective HPA 推荐
  completionStrategy:
    completionStrategyType: Periodical  # 只能是 "Once" 或 "Periodical"，Once 表示一次， Periodical 表示周期性进行
    periodSeconds: 86400                # 每隔多长时间分析推荐资源，单位为秒，此处表示一天一次
  resourceSelectors:                    # 选择要进行分析和推荐的资源对象
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
  type: Resource                       # 只能是 "Resource" 或 "HPA"，Resource 表示资源推荐，HPA 表示 Effective HPA 推荐
  completionStrategy:
    completionStrategyType: Periodical # 只能是 "Once" 或 "Periodical"，Once 表示一次， Periodical 表示周期性进行
    periodSeconds: 3600                # 每隔多长时间分析推荐资源，单位为秒，此处表示一小时一次
  resourceSelectors:                   # 选择要进行分析和推荐的资源对象
    - kind: Deployment
      apiVersion: apps/v1
      name: metric-adapter


```

上述命令会在集群里面生成两个名为：`craned-resource` 和`metric-adapter-resource` 的 `Analytics` 的资源对象，即在原来的工作负载名后加了 `-resource` 字符串。

```bash
# 执行如下命令：
kubectl get analytics -n crane-system

# 结果如下：
NAME                      AGE
craned-resource           15m
metric-adapter-resource   15m
```

3. 您可以通过查看下述命令查看 `craned` 的资源用量推荐值所在的资源对象名称：

```bash
kubectl get analytics craned-resource -n crane-system -o yaml
```

输出结果为：

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
  - name: craned-resource-resource-j7shb	# 此处为资源对象名称
    namespace: crane-system
    uid: 8ce2eedc-7969-4b80-8aee-fd4a98d6a8b6    
```

4. 具体的推荐值时存在 `Recommend` 类型的资源对象中，通过上述命令获得了资源对象名称之后，您可以通过以下命令查看具体的资源推荐值：

```bash
kubectl get recommend -n crane-system craned-resource-resource-j7shb -o yaml
```

输出结果为：

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
  resourceRequest:	# 此处为 Crane 所计算出的资源推荐值
    containers:
    - containerName: craned
      target:
        cpu: 114m
        memory: 120586239m
```

关于使用资源推荐的注意事项：

* 资源推荐基于历史普罗米修斯指标进行计算、分析和建议。
* 使用 **[Percentile](https://en.wikipedia.org/wiki/Percentile)**  算法，和 [VPA](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler) 的原理类似。
* 如果工作负载长时间运行，例如几周，结果将更准确。

## Effective HPA 推荐使用方式

1. [部署 Crane](https://docs.gocrane.io/dev/zh/installation/)，当相关的资源对象成功运行时，表示 Crane 的组件已部署完成。如下所示：

   ```shell
   NAME                            READY   UP-TO-DATE   AVAILABLE   AGE
   craned                          1/1     1            1           31m
   fadvisor                        1/1     1            1           41m
   grafana                         1/1     1            1           42m
   metric-adapter                  1/1     1            1           31m
   prometheus-kube-state-metrics   1/1     1            1           43m
   prometheus-server               1/1     1            1           43m
   ```

2. 在集群里部署“Analytics”类型资源对象，作用在名为 `craned` and `metric-adapter` 的 Deployment 上：

```bash
kubectl apply -f https://raw.githubusercontent.com/gocrane/crane/main/examples/analytics/analytics-hpa.yaml
kubectl get analytics -n crane-system 
```

```yaml title="analytics-hpa.yaml" hl_lines="7 24 11-14 28-31"
# 上述命令操作的具体 YAML 如下：
apiVersion: analysis.crane.io/v1alpha1
kind: Analytics
metadata:
  name: craned-hpa
  namespace: crane-system
spec:
  type: HPA                        # 只能是 "Resource" 或 "HPA"，Resource 表示资源推荐，HPA 表示 Effective HPA 推荐
  completionStrategy:
    completionStrategyType: Periodical  # 只能是 "Once" 或 "Periodical"，Once 表示一次， Periodical 表示周期性进行
    periodSeconds: 600                  # 每隔多长时间分析推荐资源，单位为秒，此处表示10分钟一次
  resourceSelectors:                    # 选择要进行分析和推荐的资源对象
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
  type: HPA                       # 只能是 "Resource" 或 "HPA"，Resource 表示资源推荐，HPA 表示 Effective HPA 推荐
  completionStrategy:
    completionStrategyType: Periodical # 只能是 "Once" 或 "Periodical"，Once 表示一次， Periodical 表示周期性进行
    periodSeconds: 3600                # 每隔多长时间分析推荐资源，单位为秒，此处表示一小时一次
  resourceSelectors:                   # 选择要进行分析和推荐的资源对象
    - kind: Deployment
      apiVersion: apps/v1
      name: metric-adapter
```


上述命令会在集群里面生成两个名为：`craned-hpa` 和`metric-adapter-hpa` 的 `Analytics` 的资源对象，即在原来的工作负载名后加了 `-hpa` 字符串。

```bash
# 执行如下命令：
kubectl get analytics -n crane-system

# 结果如下：
NAME                      AGE
craned-hpa          15m
metric-adapter-hpa   15m

```

3. 您可以通过查看下述命令查看 `craned` 的 Effective HPA 推荐值所在的资源对象名称：

```bash
kubectl get analytics craned-hpa -n crane-system -o yaml
```

输出结果为：

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
    name: craned-hpa-hpa-2f22w	# 此处为资源对象名称
    namespace: crane-system
    uid: 397733ee-986a-4630-af75-736d2b58bfac
```

4. 具体的推荐值时存在 `Recommend` 类型的资源对象中，通过上述命令获得了资源对象名称之后，您可以通过以下命令查看具体的资源推荐值：

```bash
kubectl get recommend -n crane-system craned-resource-resource-2f22w -o yaml
```

输出结果为：

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

 `status.resourceRequest` 是推荐的 Effective HPA 的相关配置。

注意：这里报错的原因是示例的工作负载运行时间过短。

关于使用 Effective HPA 推荐的注意事项：

* Effective HPA 推荐基于历史普罗米修斯指标进行计算、分析和建议。
* 使用 **[DSP](https://en.wikipedia.org/wiki/Digital_signal_processing)** 算法来处理指标。
* 建议使用 Effective HorizontalPodAutoscaler 执行自动缩放，您可以查看 [本文档](using-time-series-prediction.md)以了解更多信息。
* 工作负载需要符合以下条件：
  * 至少一个就绪的 Pod
  * 就绪 Pod 比率应大于 50%
  * 必须填写 Pod 的 CPU Request
  * 工作负载应至少运行 **一周**，以获得足够的指标进行预测
  * 工作负载的 CPU 负载量应该是可预测的，**太低**或**太不稳定**工作负载通常是不可预测的
