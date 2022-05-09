# EffectiveHorizontalPodAutoscaler

EffectiveHorizontalPodAutoscaler（简称 EHPA）是 Crane 提供的弹性伸缩产品，它基于社区 HPA 做底层的弹性控制，支持更丰富的弹性触发策略（预测，观测，周期），让弹性更加高效，并保障了服务的质量。
 
- 提前扩容，保证服务质量：通过算法预测未来的流量洪峰提前扩容，避免扩容不及时导致的雪崩和服务稳定性故障。
- 减少无效缩容：通过预测未来可减少不必要的缩容，稳定工作负载的资源使用率，消除突刺误判。
- 支持 Cron 配置：支持 Cron-based 弹性配置，应对大促等异常流量洪峰。
- 兼容社区：使用社区 HPA 作为弹性控制的执行层，能力完全兼容社区。

## 产品功能

一个简单的 EHPA yaml 文件如下：

```yaml
apiVersion: autoscaling.crane.io/v1alpha1
kind: EffectiveHorizontalPodAutoscaler
metadata:
  name: php-apache
spec:
  scaleTargetRef: #(1)
    apiVersion: apps/v1
    kind: Deployment
    name: php-apache
  minReplicas: 1 #(2)
  maxReplicas: 10 #(3)
  scaleStrategy: Auto #(4)
  metrics: #(5)
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 50
  prediction: #(6)
    predictionWindowSeconds: 3600 #(7)
    predictionAlgorithm:
      algorithmType: dsp
      dsp:
        sampleInterval: "60s"
        historyLength: "3d"
```

1. ScaleTargetRef 配置你希望弹性的工作负载。
2. MinReplicas 指定了自动缩容的最小值。
3. MaxReplicas 指定了自动扩容的最大值。
4. ScaleStrategy 定义了弹性的策略，值可以是 "Auto" and "Preview".
5. Metrics 定义了弹性阈值配置。
6. Prediction 定义了预测算法配置。
7. PredictionWindowSeconds 指定往后预测多久的数据。

### 基于预测的弹性

大多数在线应用的负载都有周期性的特征。我们可以根据按天或者按周的趋势预测未来的负载。EHPA 使用 DSP 算法来预测应用未来的时间序列数据。

以下是一个开启了预测能力的 EHPA 模版例子：
```yaml
apiVersion: autoscaling.crane.io/v1alpha1
kind: EffectiveHorizontalPodAutoscaler
spec:
  prediction:
    predictionWindowSeconds: 3600
    predictionAlgorithm:
      algorithmType: dsp
      dsp:
        sampleInterval: "60s"
        historyLength: "3d"

```

#### 监控数据兜底

在使用预测算法预测时，你可能会担心预测数据不准带来一定的风险，EHPA 在计算副本数时，不仅会按预测数据计算，同时也会考虑实际监控数据来兜底，提升了弹性的安全性。
实现的原理是当你在 EHPA 中定义 `spec.metrics` 并且开启弹性预测时，EffectiveHPAController 会在创建底层管理的 HPA 时按策略自动生成多条 Metric Spec。

例如，当用户在 EHPA 的 yaml 里定义如下 Metric Spec：
```yaml
apiVersion: autoscaling.crane.io/v1alpha1
kind: EffectiveHorizontalPodAutoscaler
spec:
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 50
```

它会自动转换成两条 HPA 的阈值配置：
```yaml
apiVersion: autoscaling/v2beta1
kind: HorizontalPodAutoscaler
spec:
  metrics:
    - pods:
        metric:
          name: crane_pod_cpu_usage
            selector:
              matchLabels:
                autoscaling.crane.io/effective-hpa-uid: f9b92249-eab9-4671-afe0-17925e5987b8
        target:
          type: AverageValue
          averageValue: 100m
      type: Pods
    - resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 50
      type: Resource
```

在上面这个例子中，用户在 EHPA 创建的 Metric 阈值配置会自动转换成底层 HPA 上的两条 Metric 阈值配置：预测 Metric 阈值和实际监控 Metric 阈值

* **预测 Metric 阈值** 是一个 custom metric。值通过 Crane 的 MetricAdapter 提供。
* **实际监控 Metric 阈值**是一个 resource metric，它和用户在 EHPA 上定义的一样。这样 HPA 会根据应用实际监控的 Metric 计算副本数。

HPA 在配置了多个弹性 Metric 阈值时，在计算副本数时会分别计算每条 Metric 对应的副本数，并选择**最大**的那个副本数作为最终的推荐弹性结果。

#### 水平弹性的执行流程

1. EffectiveHPAController 创建 HorizontalPodAutoscaler 和 TimeSeriesPrediction 对象 
2. PredictionCore 从 prometheus 获取历史 metric 通过预测算法计算，将结果记录到 TimeSeriesPrediction
3. HPAController 通过 metric client 从 KubeApiServer 读取 metric 数据
4. KubeApiServer 将请求路由到 Crane 的 MetricAdapter。
5. HPAController 计算所有的 Metric 返回的结果得到最终的弹性副本推荐。
6. HPAController 调用 scale API 对目标应用扩/缩容。

整体流程图如下：
![crane-ehpa](../images/crane-ehpa.png)

#### 用户案例
我们通过一个生产环境的客户案例来介绍 EHPA 的落地效果。

我们将生产上的数据在预发环境重放，对比使用 EHPA 和社区的 HPA 的弹性效果。

下图的红线是应用在一天内的实际 CPU 使用量曲线，我们可以看到在8点，12点，晚上8点时是使用高峰。绿线是 EHPA 预测的 CPU 使用量。
![craen-ehpa-metrics-chart](../images/crane-ehpa-metrics-chart.png)

下图是对应的自动弹性的副本数曲线，红线是社区 HPA 的副本数曲线，绿线是 EHPA 的副本数曲线。
![crane-ehpa-metrics-replicas-chart](../images/crane-ehpa-replicas-chart.png)

可以看到 EHPA 具有以下优势：

* 在流量洪峰来临前扩容。
* 当流量先降后立刻升时不做无效缩容。
* 相比 HPA 更少的弹性次数却更高效。

### ScaleStrategy 弹性策略
EHPA 提供了两种弹性策略：`Auto` 和 `Preview`。用户可以随时切换它并立即生效。

#### Auto
Auto 策略下 EHPA 会自动执行弹性行为。默认 EHPA 的策略是 Auto。在这个模式下 EHPA 会创建一个社区的 HPA 对象并自动接管它的生命周期。我们不建议用户修改或者控制这个底层的 HPA 对象，当 EHPA 被删除时，底层的 HPA 对象也会一并删除。

#### Preview
Preview 策略提供了一种让 EHPA 不自动执行弹性的能力。所以你可以通过 EHPA 的 desiredReplicas 字段观测 EHPA 计算出的副本数。用户可以随时在两个模式间切换，当用户切换到 Preview 模式时，用户可以通过 `spec.specificReplicas` 调整应用的副本数，如果 `spec.specificReplicas` 为空，则不会对应用执行弹性，但是依然会执行副本数的计算。

以下是一个配置成 Preview 模式的 EHPA 模版例子：
```yaml
apiVersion: autoscaling.crane.io/v1alpha1
kind: EffectiveHorizontalPodAutoscaler
spec:
  scaleStrategy: Preview   # ScaleStrategy indicate the strategy to scaling target, value can be "Auto" and "Preview".
  specificReplicas: 5      # SpecificReplicas specify the target replicas.
status:
  expectReplicas: 4        # expectReplicas is the calculated replicas that based on prediction metrics or spec.specificReplicas.
  currentReplicas: 4       # currentReplicas is actual replicas from target
```

### HorizontalPodAutoscaler 社区兼容
EHPA 从设计之出就希望和社区的 HPA 兼容，因为我们不希望重新造一个类似 HPA 的轮子，HPA 在不断演进的过程已经解决了很多通用的问题，EHPA 希望在 HPA 的基础上提供更高阶的 CRD，EHPA 的功能是社区 HPA 的超集。

EHPA 也会持续跟进支持 HPA 的新功能。

### EffectiveHorizontalPodAutoscaler status
EHPA 的 Status 包括了自身的 Status 同时也汇聚了底层 HPA 的部分 Status。

以下是一个 EHPA 的 Status yaml例子：
```yaml
apiVersion: autoscaling.crane.io/v1alpha1
kind: EffectiveHorizontalPodAutoscaler
status:
  conditions:                                               
  - lastTransitionTime: "2021-11-30T08:18:59Z"
    message: the HPA controller was able to get the target's current scale
    reason: SucceededGetScale
    status: "True"
    type: AbleToScale
  - lastTransitionTime: "2021-11-30T08:18:59Z"
    message: Effective HPA is ready
    reason: EffectiveHorizontalPodAutoscalerReady
    status: "True"
    type: Ready
  currentReplicas: 1
  expectReplicas: 0

```

## 常见问题

### 错误: unable to get metric crane_pod_cpu_usage

当你查看 EffectiveHorizontalPodAutoscaler 的 Status 时，可以会看到这样的错误：

```yaml
- lastTransitionTime: "2022-05-15T14:05:43Z"
  message: 'the HPA was unable to compute the replica count: unable to get metric
    crane_pod_cpu_usage: unable to fetch metrics from custom metrics API: TimeSeriesPrediction
    is not ready. '
  reason: FailedGetPodsMetric
  status: "False"
  type: ScalingActive
```

原因：不是所有的工作负载的 CPU 使用率都是可预测的，当无法预测时就会显示以上错误。
reason: Not all workload's cpu metric are predictable, if predict your workload failed, it will show above errors.

解决方案：
- 等一段时间再看。预测算法 `DSP` 需要一定时间的数据才能进行预测。希望了解算法细节的可以查看算法的文档。
- EffectiveHorizontalPodAutoscaler 提供一种保护机制，当预测失效时依然能通过实际的 CPU 使用率工作。