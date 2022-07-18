# Qos Ensurance
QoS ensurance guarantees the stability of the pods running on Kubernetes.
Disable schedule, throttle, evict will be applied to low priority pods when the higher priority pods is impacted by resource competition.

## Qos Ensurance Architecture
Qos ensurance's architecture is shown as below. It contains three modules.

1. state collector: collect metrics periodically
2. anomaly analyzer: analyze the node triggered anomaly used collected metrics
3. action executor: execute avoidance actions, include disable scheduling, throttle and eviction.

![crane-qos-enurance](../images/crane-qos-ensurance.png)

The main process:

1. State collector synchronizes policies from kube-apiserver.
2. If the policies are changed, the state collector updates the collectors.
3. State collector collects metrics periodically.
4. State collector transmits metrics to anomaly analyzer.
5. Anomaly analyzer ranges all rules to analyze the avoidance threshold or the restored threshold reached.
6. Anomaly analyzer merges the analyzed results and notices the avoidance actions.
7. Action executor executes actions based on the analyzed results.

## Disable Scheduling

The following AvoidanceAction and NodeQOSEnsurancePolicy can be defined. As a result, when the node CPU usage triggers the threshold, disable schedule action for the node will be executed.

The sample YAML looks like below:

```yaml
apiVersion: ensurance.crane.io/v1alpha1
kind: AvoidanceAction
metadata:
  labels:
    app: system
  name: disablescheduling
spec:
  description: disable schedule new pods to the node
  coolDownSeconds: 300  # The minimum wait time of the node from  scheduling disable status to normal status
```

```yaml
apiVersion: ensurance.crane.io/v1alpha1
kind: NodeQOSEnsurancePolicy
metadata:
  name: "waterline1"
  labels:
    app: "system"
spec:
  nodeQualityProbe: 
    timeoutSeconds: 10
    nodeLocalGet:
      localCacheTTLSeconds: 60
  objectiveEnsurances:
  - name: "cpu-usage"
    avoidanceThreshold: 2 #(1) 
    restoreThreshold: 2 #(2)
    actionName: "disablescheduling" #(3) 
    strategy: "None" #(4) 
    metricRule:
      name: "cpu_total_usage" #(5) 
      value: 4000 #(6) 
```

1. We consider the rule is triggered, when the threshold reached continued so many times
2. We consider the rule is restored, when the threshold not reached continued so many times
3. Name of AvoidanceAction which be associated
4. Strategy for the action, you can set it "Preview" to not perform actually
5. Name of metric
6. Threshold of metric

Please check the video to learn more about the scheduling disable actions.

<script id="asciicast-480735" src="https://asciinema.org/a/480735.js" async></script>

## Throttle

The following AvoidanceAction and NodeQOSEnsurancePolicy can be defined. As a result, when the node CPU usage triggers the threshold, throttle action for the node will be executed.

The sample YAML looks like below:

```yaml
apiVersion: ensurance.crane.io/v1alpha1
kind: AvoidanceAction
metadata:
  name: throttle
  labels:
    app: system
spec:
  coolDownSeconds: 300
  throttle:
    cpuThrottle:
      minCPURatio: 10 #(1)
      stepCPURatio: 10 #(2) 
  description: "throttle low priority pods"
```

1. The minimal ratio of the CPU quota, if the pod is throttled lower than this ratio, it will be set to this.
2. The step for throttle action. It will reduce this percentage of CPU quota in each avoidance triggered.It will increase this percentage of CPU quota in each restored.

```yaml
apiVersion: ensurance.crane.io/v1alpha1
kind: NodeQOSEnsurancePolicy
metadata:
  name: "waterline2"
  labels:
    app: "system"
spec:
  nodeQualityProbe:
    timeoutSeconds: 10
    nodeLocalGet:
      localCacheTTLSeconds: 60
  objectiveEnsurances:
    - name: "cpu-usage"
      avoidanceThreshold: 2
      restoredThreshold: 2
      actionName: "throttle"
      strategy: "None"
      metricRule:
        name: "cpu_total_usage"
        value: 6000
```

## Eviction

The following YAML is another case, low priority pods on the node will be evicted, when the node CPU usage trigger the threshold.

```yaml
apiVersion: ensurance.crane.io/v1alpha1
kind: AvoidanceAction
metadata:
  name: eviction
  labels:
    app: system
spec:
  coolDownSeconds: 300
  eviction:
    terminationGracePeriodSeconds: 30 #(1) 
  description: "evict low priority pods"
```

1. Duration in seconds the pod needs to terminate gracefully.

```yaml
apiVersion: ensurance.crane.io/v1alpha1
kind: NodeQOSEnsurancePolicy
metadata:
  name: "waterline3"
  labels:
    app: "system"
spec:
  nodeQualityProbe: 
    timeoutSeconds: 10
    nodeLocalGet:
      localCacheTTLSeconds: 60
  objectiveEnsurances:
  - name: "cpu-usage"
    avoidanceThreshold: 2
    restoreThreshold: 2
    actionName: "eviction"
    strategy: "Preview" #(1) 
    metricRule:
      name: "cpu_total_usage"
      value: 6000
```

1. Strategy for the action, "Preview" to not perform actually

## Supported Metrics

Name     | Description
---------|-------------
cpu_total_usage | node cpu usage
cpu_total_utilization | node cpu utilization
