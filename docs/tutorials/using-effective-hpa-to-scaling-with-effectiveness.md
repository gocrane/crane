# Use EffectiveHorizontalPodAutoscaler to scaling applications with effectiveness

EffectiveHorizontalPodAutoscaler provides advanced functions to help you manage scaling for applications easier. It is compatible with
HorizontalPodAutoscaler and extends more useful features to reduce the usage difficulty。

EffectiveHorizontalPodAutoscaler support prediction-driven autoscaling, through this ability user can forecast the peak flow and scale up their
application before it, also user can know when the peak flow will gone and scale down their application slowly.

Besides that EffectiveHorizontalPodAutoscaler defines several scale strategies to support various scaling scenes.

# Features
A sample EffectiveHorizontalPodAutoscaler yaml looks like following:
```yaml
apiVersion: autoscaling.crane.io/v1alpha1
kind: EffectiveHorizontalPodAutoscaler
metadata:
  name: php-apache
spec:
  # ScaleTargetRef is the reference to the workload that should be scaled.
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: php-apache
  minReplicas: 1        # MinReplicas is the lower limit replicas to the scale target which the autoscaler can scale down to.
  maxReplicas: 10       # MaxReplicas is the upper limit replicas to the scale target which the autoscaler can scale up to.
  scaleStrategy: Auto   # ScaleStrategy indicate the strategy to scaling target, value can be "Auto" and "Manual".
  # Metrics contains the specifications for which to use to calculate the desired replica count.
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 50
  # Prediction defines configurations for predict resources.
  # If unspecified, defaults don't enable prediction.
  prediction:
    predictionWindowSeconds: 3600   # PredictionWindowSeconds is the time window to predict metrics in the future.
    predictionAlgorithm:
      algorithmType: dsp
      dsp:
        sampleInterval: "60s"
        historyLength: "3d"


```

* spec.scaleTargetRef defines the reference to the workload that should be scaled.
* spec.minReplicas is the lower limit replicas to the scale target which the autoscaler can scale down to.
* spec.maxReplicas is the upper limit replicas to the scale target which the autoscaler can scale up to.
* spec.metrics indicate the strategy to scaling target, value can be "Auto" and "Manual".
* spec.metrics contains the specifications for which to use to calculate the desired replica count. Please refer to the details:
* spec.prediction defines configurations for predict resources.If unspecified, defaults don't enable prediction.

## Prediction-driven autoscaling
Most of online applications have regular pattern, we can use algorithm to predict future values in hours or days. DSP is a time series prediction algorithm that
can works for prediction application metrics.

The following shows a sample EffectiveHorizontalPodAutoscaler yaml which contains prediction configuration.
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

### Metric conversion
When user defines a `spec.metrics` in EffectiveHorizontalPodAutoscaler and prediction configuration is enabled, EffectiveHPAController will convert it to a new metrics
and configuration the background HorizontalPodAutoscaler. Let's use a sample to demonstrate it.

This is a EffectiveHorizontalPodAutoscaler metrics yaml.
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

This is the converted HorizontalPodAutoscaler metrics yaml.
```yaml
apiVersion: autoscaling/v2beta1
kind: HorizontalPodAutoscaler
spec:
  metrics:
    - pods:
        metricName: pod_cpu_usage
        selector:
          matchLabels:
            autoscaling.crane.io/effective-hpa-uid: f9b92249-eab9-4671-afe0-17925e5987b8
        targetAverageValue: 100m
      type: Pods
    - resource:
        name: cpu
        targetAverageUtilization: 50
      type: Resource
```

In this sample the resource metric defined by user conversion into two metrics: prediction metric and origin metric .
* **prediction metric** is a custom metrics that provided by component MetricAdapter. Since custom metric not support `targetAverageUtilization`, we convert to `targetAverageValue` based on target pod' cpu request.
* **origin metric** is equals to user defined metrics in EffectiveHorizontalPodAutoscaler, we use this metric to defense abnormal situation like prediction metric empty or too low.

HorizontalPodAutoscaler will evaluate each metric, and propose a new scale based on that metric. The **largest** of the proposed scales will be used as the new scale.

### Horizontal scaling process
The prediction and scaling progress have six steps:
1. EffectiveHPAController create HorizontalPodAutoscaler and TimeSeriesPrediction instance 
2. PredictionCore get historic metric from prometheus and persist into TimeSeriesPrediction
3. HPAController read metrics from KubeApiServer
4. KubeApiServer forward requests to MetricAdapter and MetricServer
5. HPAController calculate all metric results and propose a new scale replicas for target
6. HPAController scale target with Scale Api

The above is the complete process flow.
<div align="center"><img src="../images/crane-ehpa.png" style="width:900px;" /></div>

### Use case
We will show a use case that using EffectiveHorizontalPodAutoscaler in production cluster.

This is an application deployment running in production. Its total cpu usage is regular that in midday evening morning cpu usage is higher and in midnight
 cpu usage is lower. the following chart shows the actual cpu usage for red line and the prediction cpu usage for green line.
<div align="center"><img src="../images/crane-ehpa-metrics-chart.png" style="width:900px;" /></div>

Let's see a comparison from EffectiveHorizontalPodAutoscaler and HorizontalPodAutoscaler below.
The red line is the replica number curve when using HorizontalPodAutoscaler and the green line is the result the replica number curve when using EffectiveHorizontalPodAutoscaler.
<div align="center"><img src="../images/crane-ehpa-replicas-chart.png" style="width:900px;" /></div>

we can see the benefit from above chart:
* scale up before peek flow
* scale down slowly after peek flow
* less replicas changes compares to HorizontalPodAutoscaler

## ScaleStrategy
EffectiveHorizontalPodAutoscaler provides two scale strategy: `Auto` and `Manual`, user can change the scale strategy in runtime, and it will take effect at once.

### Auto
Auto strategy means execution scaling based on metrics, this is the default strategy. In this strategy EffectiveHorizontalPodAutoscaler will create and control a HorizontalPodAutoscaler 
instance in background, we recommend not configuration the background HorizontalPodAutoscaler because any unexpected change in HorizontalPodAutoscaler will be adjusted by EffectiveHPAController。
If user delete EffectiveHorizontalPodAutoscaler, the background HorizontalPodAutoscaler will be cleanup too.

### Manual
Manual strategy means user can specific target replicas without any automation. Sometimes user want to disable autoscaling and control the target immediately，
they can apply `spec.scaleStrategy` to `Manual`, then EffectiveHPAController will disable HorizontalPodAutoscaler if exists and scale the target to the value 
`spec.specificReplicas`, if user not set `spec.specificReplicas`, when ScaleStrategy is change to Manual, it will just stop scaling.

A sample manual configuration looks like following:
```yaml
apiVersion: autoscaling.crane.io/v1alpha1
kind: EffectiveHorizontalPodAutoscaler
spec:
  scaleStrategy: Auto   # ScaleStrategy indicate the strategy to scaling target, value can be "Auto" and "Manual".
  specificReplicas: 5   # SpecificReplicas specify the target replicas.

```

## HorizontalPodAutoscaler compatible
EffectiveHorizontalPodAutoscaler is designed to be compatible with k8s native HorizontalPodAutoscaler, because we don't reinvent the autoscaling part but take advantage of the extension
from HorizontalPodAutoscaler and build a high level autoscaling CRD. EffectiveHorizontalPodAutoscaler support all abilities from HorizontalPodAutoscaler like metricSpec and behavior.

We will continue support incoming new feature from HorizontalPodAutoscaler.

## EffectiveHorizontalPodAutoscaler status
This is a yaml from EffectiveHorizontalPodAutoscaler.Status
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
