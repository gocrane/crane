# Qos Ensurance
Qos Ensurance 保证了运行在 Kubernetes 上的 Pod 的稳定性。
当较高优先级的 Pod 受到资源竞争的影响时，Disable Schedule、Throttle以及Evict 将应用于低优先级的 Pod。


## Qos Ensurance 架构
Qos ensurance 的架构如下图所示。它包含三个模块。

1. `State Collector`：定期收集指标
2. `Anomaly Analyzer`：使用收集指标，以分析节点是否发生异常
3. `Action Executor`：执行回避动作，包括 Disable Scheduling、Throttle 和 Eviction。

![crane-qos-enurance](../images/crane-qos-ensurance.png)

主要流程：

1. `State Collector` 从 kube-apiserver 同步策略。
2. 如果策略发生更改，`State Collector`会更新指标收集规则。
3. `State Collector`定期收集指标。
4. `State Collector`将指标传输到`Anomaly Analyzer`。
5. `Anomaly Analyzer`对所有规则进行范围分析，以分析达到的回避阈值或恢复阈值。
6. `Anomaly Analyzer`合并分析结果并通知`Action Executor`执行回避动作。
7. `Action Executor`根据分析结果执行动作。

## Disable Scheduling

定义 `AvoidanceAction` 和 `NodeQOSEnsurancePolicy`。

当节点 CPU 使用率触发回避阈值时，将该节点设置为禁用调度。


示例 YAML 如下所示：

```yaml title="AvoidanceAction"
apiVersion: ensurance.crane.io/v1alpha1
kind: AvoidanceAction
metadata:
  labels:
    app: system
  name: disablescheduling
spec:
  description: disable schedule new pods to the node
  coolDownSeconds: 300  #(1) 
```

1. 节点从禁止调度状态到正常状态的最小等待时间
 
```yaml title="NodeQOSEnsurancePolicy"
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

1. 当达到阈值并持续多次，那么我们认为规则被触发
2. 当阈值未达到并继续多次, 那么我们认为规则已恢复
3. 关联到 AvoidanceAction 名称
4. 动作的策略，你可以将其设置为“预览”以不实际执行
5. 指标名称
6. 指标的阈值

请观看视频以了解更多`Disable Scheduling`的细节。

<script id="asciicast-480735" src="https://asciinema.org/a/480735.js" async></script>

## Throttle 

定义 `AvoidanceAction` 和 `NodeQOSEnsurancePolicy`。

当节点 CPU 使用率触发回避阈值时，将执行节点的`Throttle Action`。

示例 YAML 如下所示：

```yaml title="AvoidanceAction"
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

1. CPU 配额的最小比例，如果 pod 被限制低于这个比例，就会被设置为这个。
2. 该配置设置给`Throttle Action`。它将在每个触发的回避动作中减少这个 CPU 配额占比。它会在每个恢复动作中增加这个 CPU 配额占比。

```yaml title="NodeQOSEnsurancePolicy"
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

下面的 YAML 是另一种情况，当节点 CPU 使用率触发阈值时，节点上的低优先级 pod 将被驱逐。

```yaml title="AvoidanceAction"
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

1. pod 需要优雅终止的持续时间（以秒为单位）。

```yaml title="NodeQOSEnsurancePolicy"
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
    actionName: "evict"
    strategy: "Preview" #(1) 
    metricRule:
      name: "cpu_total_usage"
      value: 6000
```

1. 回避动作策略。当设置为`Preview`时，将不会被实际执行

## Supported Metrics

Name     | Description
---------|-------------
cpu_total_usage | node cpu usage
cpu_total_utilization | node cpu utilization
