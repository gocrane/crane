---
title: "TimeSeriesPrediction"
description: "TimeSeriesPrediction 功能介绍"
weight: 24
---

Knowing the future makes things easier for us.

---

许多业务在时间序列上天然存在周期性的，尤其是对于那些直接或间接为“人”服务的业务。这种周期性是由人们日常活动的规律性决定的。例如，人们习惯于中午和晚上点外卖；早晚总有交通高峰；即使是搜索等模式不那么明显的服务，夜间的请求量也远低于白天时间。对于这类业务相关的应用来说，从过去几天的历史数据中推断出次日的指标，或者从上周一的数据中推断出下周一的访问量是很自然的想法。通过预测未来 24 小时内的指标或流量模式，我们可以更好地管理我们的应用程序实例，稳定我们的系统，同时降低成本。

`TimeSeriesPrediction` 被用于预测 Kubernetes 对象指标。它基于 `PredictionCore` 进行预测。

## Features
`TimeSeriesPrediction` 的示例 yaml 如下所示：

```yaml title="TimeSeriesPrediction"
apiVersion: prediction.crane.io/v1alpha1
kind: TimeSeriesPrediction
metadata:
  name: node-resource-percentile
  namespace: default
spec:
  targetRef:
    kind: Node
    name: 192.168.56.166
  predictionWindowSeconds: 600
  predictionMetrics:
    - resourceIdentifier: node-cpu
      type: ResourceQuery
      resourceQuery: cpu
      algorithm:
        algorithmType: "percentile"
        percentile:
          sampleInterval: "1m"
          minSampleWeight: "1.0"
          histogram:
            maxValue: "10000.0"
            epsilon: "1e-10"
            halfLife: "12h"
            bucketSize: "10"
            firstBucketSize: "40"
            bucketSizeGrowthRatio: "1.5"
    - resourceIdentifier: node-mem
      type: ResourceQuery
      resourceQuery: memory
      algorithm:
        algorithmType: "percentile"
        percentile:
          sampleInterval: "1m"
          minSampleWeight: "1.0"
          histogram:
            maxValue: "1000000.0"
            epsilon: "1e-10"
            halfLife: "12h"
            bucketSize: "10"
            firstBucketSize: "40"
            bucketSizeGrowthRatio: "1.5"
```

* `spec.targetRef` 定义了对 Kubernetes 对象的引用，包括 Node 或其他工作负载，例如 Deployment。
* `spec.predictionMetrics` 定义了关于 `spec.targetRef` 的指标。
* `spec.predictionWindowSeconds` 是预测时间序列持续时间。`TimeSeriesPredictionController` 将轮换 `spec.Status` 中的预测数据，以供消费者使用预测的时间序列数据。

## Prediction Metrics
```yaml title="TimeSeriesPrediction"
apiVersion: prediction.crane.io/v1alpha1
kind: TimeSeriesPrediction
metadata:
  name: node-resource-percentile
  namespace: default
spec:
  predictionMetrics:
    - resourceIdentifier: node-cpu
      type: ResourceQuery
      resourceQuery: cpu
      algorithm:
        algorithmType: "percentile"
        percentile:
          sampleInterval: "1m"
          minSampleWeight: "1.0"
          histogram:
            maxValue: "10000.0"
            epsilon: "1e-10"
            halfLife: "12h"
            bucketSize: "10"
            firstBucketSize: "40"
            bucketSizeGrowthRatio: "1.5"
```

### Metric Type

现在我们只支持 `prometheus` 作为数据源。我们定义`MetricType`与数据源进行结合。但是现在可能有些数据源不支持 `MetricType`。

指标查询有以下三种类型：

- `ResourceQuery`是 kubernetes 内置的资源指标，例如 cpu 或 memory。Crane目前只支持 CPU 和内存。
- `RawQuery`是通过 DSL 的查询，比如 prometheus 查询语句。现在已支持 Prometheus 。
- `ExpressionQuery`是一个表达式查询。


### Algorithm

`Algorithm`定义算法类型和参数来预测指标。现在有两种算法：

- `dsp`是一种预测时间序列的算法，它基于 FFT（快速傅里叶变换），擅长预测一些具有季节性和周期的时间序列。
- `percentile`是一种估计时间序列，并找到代表过去时间序列的推荐值的算法，它基于指数衰减权重直方图统计。它是用来估计一个时间序列的，它不擅长预测一个时间序列，虽然`percentile`可以输出一个时间序列的预测数据，但是都是一样的值。**所以如果你想预测一个时间序列，dsp 是一个更好的选择。**
 
