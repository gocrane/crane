# EffectiveHorizontalPodAutoscaler

EffectiveHorizontalPodAutoscaler helps you manage application scaling in an easy way. 

It is compatible with HorizontalPodAutoscaler but extends more features.

EffectiveHorizontalPodAutoscaler supports prediction-driven autoscaling. 

With this capability, user can forecast the incoming peak flow and scale up their application ahead, also user can know when the peak flow will end and scale down their application gracefully.

Besides that, EffectiveHorizontalPodAutoscaler also defines several scale strategies to support different scaling scenarios.

## Features
A EffectiveHorizontalPodAutoscaler sample yaml looks like below:

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

1. ScaleTargetRef is the reference to the workload that should be scaled.
2. MinReplicas is the lower limit replicas to the scale target which the autoscaler can scale down to.
3. MaxReplicas is the upper limit replicas to the scale target which the autoscaler can scale up to.
4. ScaleStrategy indicates the strategy to scaling target, value can be "Auto" and "Preview".
5. Metrics contains the specifications for which to use to calculate the desired replica count.
6. Prediction defines configurations for predict resources.If unspecified, defaults don't enable prediction.
7. PredictionWindowSeconds is the time window to predict metrics in the future.

### Params Description

* spec.scaleTargetRef defines the reference to the workload that should be scaled.
* spec.minReplicas is the lower limit replicas to the scale target which the autoscaler can scale down to.
* spec.maxReplicas is the upper limit replicas to the scale target which the autoscaler can scale up to.
* spec.scaleStrategy indicates the strategy to scaling target, value can be "Auto" and "Preview".
* spec.metrics contains the specifications for which to use to calculate the desired replica count. Please refer to the details:
* spec.prediction defines configurations for predict resources.If unspecified, defaults don't enable prediction.

### Prediction-driven autoscaling
Most of online applications follow regular pattern. We can predict future trend of hours or days. DSP is a time series prediction algorithm that applicable for application metrics prediction.

The following shows a sample EffectiveHorizontalPodAutoscaler yaml with prediction enabled.
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

#### Metric conversion
When user defines `spec.metrics` in EffectiveHorizontalPodAutoscaler and prediction configuration is enabled, EffectiveHPAController will convert it to a new metric and configure the background HorizontalPodAutoscaler. 

This is a source EffectiveHorizontalPodAutoscaler yaml for metric definition.
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

It's converted to underlying HorizontalPodAutoscaler metrics yaml.
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

In this sample, the resource metric defined by user is converted into two metrics: prediction metric and origin metric.

* **prediction metric** is custom metrics that provided by component MetricAdapter. Since custom metric doesn't support `targetAverageUtilization`, it's converted to `targetAverageValue` based on target pod cpu request.
* **origin metric** is equivalent to user defined metrics in EffectiveHorizontalPodAutoscaler, to fall back to baseline user defined in case of some unexpected situation e.g. business traffic sudden growth.

HorizontalPodAutoscaler will calculate on each metric, and propose new replicas based on that. The **largest** one will be picked as the new scale.

#### Horizontal scaling process
There are six steps of prediction and scaling process:

1. EffectiveHPAController create HorizontalPodAutoscaler and TimeSeriesPrediction instance 
2. PredictionCore get historic metric from prometheus and persist into TimeSeriesPrediction
3. HPAController read metrics from KubeApiServer
4. KubeApiServer forward requests to MetricAdapter and MetricServer
5. HPAController calculate all metric results and propose a new scale replicas for target
6. HPAController scale target with Scale Api

Below is the process flow.
![crane-ehpa](../images/crane-ehpa.png)

#### Use case
Let's take one use case that using EffectiveHorizontalPodAutoscaler in production cluster.

We did a profiling on the load history of one application in production and replayed it in staging environment. With the same application, we leverage both EffectiveHorizontalPodAutoscaler and HorizontalPodAutoscaler to manage the scale and compare the result.

From the red line in below chart, we can see its actual total cpu usage is high at ~8am, ~12pm, ~8pm and low in midnight. The green line shows the prediction cpu usage trend.
![craen-ehpa-metrics-chart](../images/crane-ehpa-metrics-chart.png)

Below is the comparison result between EffectiveHorizontalPodAutoscaler and HorizontalPodAutoscaler. The red line is the replica number generated by HorizontalPodAutoscaler and the green line is the result from EffectiveHorizontalPodAutoscaler.
![crane-ehpa-metrics-replicas-chart](../images/crane-ehpa-replicas-chart.png)

We can see significant improvement with EffectiveHorizontalPodAutoscaler:

* Scale up in advance before peek flow
* Scale down gracefully after peek flow
* Fewer replicas changes than HorizontalPodAutoscaler

### ScaleStrategy
EffectiveHorizontalPodAutoscaler provides two strategies for scaling: `Auto` and `Preview`. User can change the strategy at runtime, and it will take effect on the fly.

#### Auto
Auto strategy achieves automatic scaling based on metrics. It is the default strategy. With this strategy, EffectiveHorizontalPodAutoscaler will create and control a HorizontalPodAutoscaler instance in backend. We don't recommend explicit configuration on the underlying HorizontalPodAutoscaler because it will be overridden by EffectiveHPAController. If user delete EffectiveHorizontalPodAutoscaler, HorizontalPodAutoscaler will be cleaned up too.

#### Preview
Preview strategy means EffectiveHorizontalPodAutoscaler won't change target's replicas automatically, so you can preview the calculated replicas and control target's replicas by themselves. User can switch from default strategy to this one by applying `spec.scaleStrategy` to `Preview`. It will take effect immediately, During the switch, EffectiveHPAController will disable HorizontalPodAutoscaler if exists and scale the target to the value `spec.specificReplicas`, if user not set `spec.specificReplicas`, when ScaleStrategy is change to Preview, it will just stop scaling.

A sample preview configuration looks like following:
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

### HorizontalPodAutoscaler compatible
EffectiveHorizontalPodAutoscaler is designed to be compatible with k8s native HorizontalPodAutoscaler, because we don't reinvent the autoscaling part but take advantage of the extension from HorizontalPodAutoscaler and build a high level autoscaling CRD. EffectiveHorizontalPodAutoscaler support all abilities from HorizontalPodAutoscaler like metricSpec and behavior.

EffectiveHorizontalPodAutoscaler will continue support incoming new feature from HorizontalPodAutoscaler.

### EffectiveHorizontalPodAutoscaler status
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
