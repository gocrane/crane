# TimeSeriesPrediction

Knowing the future makes things easier for us.

---

Many businesses are naturally cyclical in time series, especially for those that directly or indirectly serve "people". This periodicity is determined by the regularity of people’s daily activities. For example, people are accustomed to ordering take-out at noon and in the evenings; there are always traffic peaks in the morning and evening; even for services that don't have such obvious patterns, such as searching, the amount of requests at night is much lower than that during business hours. For applications related to this kind of business, it is a natural idea to infer the next day's metrics from the historical data of the past few days, or to infer the coming Monday's access traffic from the data of last Monday. With predicted metrics or traffic patterns in the next 24 hours, we can better manage our application instances, stabilize our system, and meanwhile, reduce the cost.

TimeSeriesPrediction is used to forecast the kubernetes object metric. It is based on PredictionCore to do forecast.


# Features
A TimeSeriesPrediction sample yaml looks like below:
```yaml
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

* spec.targetRef defines the reference to the kubernetes object including Node or other workload such as Deployment.
* spec.predictionMetrics defines the metrics about the spec.targetRef.
* spec.predictionWindowSeconds is a prediction time series duration. the TimeSeriesPredictionController will rotate the predicted data in spec.Status for consumer to consume the predicted time series data.

## PredictionMetrics
```yaml
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

### MetricType

There are three types of the metric query:

 - `ResourceQuery` is a kubernetes built-in resource metric such as cpu or memory. crane supports only cpu and memory  now.
 - `RawQuery` is a query by DSL, such as prometheus query language. now support prometheus.
 - `ExpressionQuery` is a query by Expression selector. 

Now we only support prometheus as data source. We define the `MetricType` to orthogonal with the datasource. but now maybe some datasources do not support the metricType.

### Algorithm
`Algorithm` defines the algorithm type and the corresponding parameters. Currently, crane provides two built-in prediction algorithms:
 - `dsp` borrows some techniques from 'digital signal processing', such as FFT(Fast Fourier Transform), for periodicity identification and time series forecasting. It is good at predicting the time series that follow some cyclic patterns.
 - `percentile` is an algorithm for finding a recommended value based on the historical data. It uses an exponentially-decaying weights histogram to hold the historical statistics information. As its name implies, this algorithm gives a predicted value according to a percentile (such as 99th) of the historical values in a time series.
u
 

#### dsp params
Below 
```yaml
apiVersion: prediction.crane.io/v1alpha1
kind: TimeSeriesPrediction
metadata:
  name: test
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

#### percentile params 
