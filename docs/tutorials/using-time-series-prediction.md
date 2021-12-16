# TimeSeriesPrediction

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

There are three types of the metric query, ResourceQuery、ExpressionQuery、RawQuery

 - `ResourceQuery` is a kubernetes built-in resource metric such as cpu or memory. crane supports only cpu and memory  now.
 - `RawQuery` is a query by DSL, such as prometheus query language. now support prometheus.
 - `ExpressionQuery` is a query by Expression selector. 

Now we only support prometheus as data source. We define the `MetricType` to orthogonal with the datasource. but now maybe some datasources do not support the metricType.

### Algorithm
`Algorithm` define the algorithm type and params to do predict for the metric. Now there are two kinds of algorithms:
 - `dsp` is an algorithm to forcasting a time series, it is based on FFT(Fast Fourier Transform), it is good at predicting some time series with seasonality and periods.
 - `percentile` is an algorithm to estimate a time series, and find a recommended value to represent the past time series, it is based on exponentially-decaying weights historgram statistics. it is used to estimate a time series, it is not good at to predict a time sequences, although the percentile can output a time series predicted data, but it is all the same value. so if you want to predict a time sequences, dsp is a better choice.
 

#### dsp params

#### percentile params 