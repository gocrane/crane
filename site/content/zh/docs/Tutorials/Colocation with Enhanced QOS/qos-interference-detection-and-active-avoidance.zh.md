---
title: "干扰检测和主动回避"
description: "水位线功能"
weight: 19
---

## QOS Ensurance 架构
QOS ensurance 的架构如下图所示。它包含三个模块。

1. `State Collector`：定期收集指标
2. `Anomaly Analyzer`：使用收集指标，以分析节点是否发生异常
3. `Action Executor`：执行回避动作，包括 Disable Scheduling、Throttle 和 Eviction。

![](/images/crane-qos-ensurance.png)

主要流程：

1. `State Collector` 从 kube-apiserver 同步策略。
2. 如果策略发生更改，`State Collector`会更新指标收集规则。
3. `State Collector`定期收集指标。
4. `State Collector`将指标传输到`Anomaly Analyzer`。
5. `Anomaly Analyzer`对所有规则进行范围分析，以分析达到的回避阈值或恢复阈值。
6. `Anomaly Analyzer`合并分析结果并通知`Action Executor`执行回避动作。
7. `Action Executor`根据分析结果执行动作。

## 干扰检测和主动回避

### 相关CR
AvoidanceAction主要定义了检测到干扰后需要执行的操作，包含了Disable Scheduling, throttle, eviction等几个操作，并且定义了其相关的一些参数。

NodeQOS主要定义了指标采集方式和参数，水位线指标相关参数，以及指标异常时关联的回避操作，同时通过label selector将上面的内容关联到指定的节点。

PodQOS定义了指定pod可以被执行的AvoidanceAction，通常和NodeQOS搭配起来，从节点和pod的维度共同限制执行动作的范围，PodQOS支持的seletor包含label selector, 
还支持筛选特定QOSClass("BestEffort","Guaranteed"等)，特定Priority，特定Namespace的pod，并且之间采用与的方式关联。
 
### Disable Scheduling

定义 `AvoidanceAction`, `PodQOS`和 `NodeQOS`。

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

```yaml title="NodeQOS"
apiVersion: ensurance.crane.io/v1alpha1
kind: NodeQOS
metadata:
  name: "watermark1"
  labels:
    app: "system"
spec:
  nodeQualityProbe: 
    timeoutSeconds: 10
    nodeLocalGet:
      localCacheTTLSeconds: 60
  rules:
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

```yaml title="PodQOS"
apiVersion: ensurance.crane.io/v1alpha1
kind: PodQOS
metadata:
  name: all-elastic-pods
spec:
  allowedActions:  #(1) 
    - eviction  
  labelSelector:   #(2) 
    matchLabels:
      preemptible_job: "true"
```

1. 被该PodQOS关联的pod允许被执行的action为eviction
2. 通过label selector关联具有preemptible_job: "true"的pod

请观看视频以了解更多`Disable Scheduling`的细节。

<script id="asciicast-480735" src="https://asciinema.org/a/480735.js" async></script>

### Throttle

定义 `AvoidanceAction` 和 `NodeQOS`。

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

```yaml title="NodeQOS"
apiVersion: ensurance.crane.io/v1alpha1
kind: NodeQOS
metadata:
  name: "watermark2"
  labels:
    app: "system"
spec:
  nodeQualityProbe:
    timeoutSeconds: 10
    nodeLocalGet:
      localCacheTTLSeconds: 60
  rules:
    - name: "cpu-usage"
      avoidanceThreshold: 2
      restoredThreshold: 2
      actionName: "throttle"
      strategy: "None"
      metricRule:
        name: "cpu_total_usage"
        value: 6000
```

```yaml title="PodQOS"
apiVersion: ensurance.crane.io/v1alpha1
kind: PodQOS
metadata:
  name: all-be-pods
spec:
  allowedActions:
    - throttle
  scopeSelector:
    matchExpressions:
      - operator: In
        scopeName: QOSClass
        values:
          - BestEffort
```

### Eviction

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

pod 需要优雅终止的持续时间（以秒为单位）。

```yaml title="NodeQOS"
apiVersion: ensurance.crane.io/v1alpha1
kind: NodeQOS
metadata:
  name: "watermark3"
  labels:
    app: "system"
spec:
  nodeQualityProbe: 
    timeoutSeconds: 10
    nodeLocalGet:
      localCacheTTLSeconds: 60
  rules:
  - name: "cpu-usage"
    avoidanceThreshold: 2
    restoreThreshold: 2
    actionName: "eviction"
    strategy: "Preview" #(1) 
    metricRule:
      name: "cpu_total_usage"
      value: 6000
```

```yaml title="PodQOS"
apiVersion: ensurance.crane.io/v1alpha1
kind: PodQOS
metadata:
  name: all-elastic-pods
spec:
  allowedActions:   
    - eviction  
  labelSelector:  
    matchLabels:
      preemptible_job: "true"
```

回避动作策略。当设置为`Preview`时，将不会被实际执行

### 支持的水位线指标
Name     | Description
---------|-------------
cpu_total_usage | node cpu usage
cpu_total_utilization | node cpu utilization percent
memory_total_usage | node mem usage
memory_total_utilization| node mem utilization percent

具体可以参考examples/ensurance下的例子

### 与弹性资源搭配使用
为了避免主动回避操作对于高优先级业务的影响，比如误驱逐了重要业务，建议使用PodQOS关联使用了弹性资源的workload，这样在执行动作的时候只会影响这些使用了空闲资源的workload，
保证了节点上的核心业务的稳定。

弹性资源的内容可以参见[弹性资源超卖和限制](/zh-cn/docs/tutorials/colocation-with-enhanced-qos/qos-dynamic-resource-oversold-and-limit.zh)。