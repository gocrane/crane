# Qos Ensurance
Qos Ensurance 保证了运行在 Kubernetes 上的 Pod 的稳定性。

具有干扰检测和主动回避能力，当较高优先级的 Pod 受到资源竞争的影响时，Disable Schedule、Throttle以及Evict 将应用于低优先级的 Pod，支持自定义指标干扰检测和自定义操作；

同时具备增强的旁路cpuset管理能力，在绑核的同时提升资源利用效率。

具有预测算法增强的动态资源超卖能力，将空闲资源复用起来，同时结合crane的预测能力，更好地复用闲置资源。同时具有弹性资源限制功能，限制复用空闲资源的workload。

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

## 干扰检测和主动回避
### Disable Scheduling

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

### Throttle 

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

回避动作策略。当设置为`Preview`时，将不会被实际执行

### Supported Metrics

Name     | Description
---------|-------------
cpu_total_usage | node cpu usage
cpu_total_utilization | node cpu utilization

### 精确执行回避动作
通过如下两点进行，避免了对于低优pod的过度操作的同时能够更快地降低指标到指定水位线的差距，保障高优业务不受影响
1. 排序pod


   crane实现了一些通用的排序方法（之后会更多地完善）：

   classAndPriority： 比较两个pod的QOSClass和class value，优先比较QOSClass，再比较class value；priority高的排在后面优先级更高

   runningTime：比较两个pod的运行时间，运行时间长的排在后面优先级更高

   如果仅需使用这两个排序策略，使用默认的排序方法即可：会首先比较pod的优先级，之后比较pod对应指标的用量，之后比较pod的运行时长，有一个维度可以比较出结果即为pod的排序结果

   以cpu usage指标的排序为例，还扩展了一些与自身指标相关的排序策略， 如cpu usage 使用量的排序，会依次比较两个pod的优先级，如果优先级相同的情况下，再比较cpu用量，如果cpu用量也相同的情况下继续比较扩展cpu资源用量, 最后比较pod的运行时长，当某一个指标存在差异时即可返回比较结果：`orderedBy(classAndPriority, cpuUsage, extCpuUsage, runningTime).Sort(pods)`


2. 参考水位线和pod用量执行回避动作
   ```go
   //将所有触发水位线的metrics根据其Quantified属性区分为两部分
   metricsQuantified, MetricsNotQuantified := ThrottleDownWaterLine.DivideMetricsByQuantified()
   // 如果存在不可Quantified的metric，获取具有最高ActionPriority的一个throttleAble的metric对所选择的所有pod进行操作
   if len(MetricsNotThrottleQuantified) != 0 {
       highestPrioriyMetric := GetHighestPriorityThrottleAbleMetric()
       t.throttlePods(ctx, &totalReleased, highestPrioriyMetric)
   } else {
       //获取节点和workload的最新用量，构造和水位线差距
       ThrottoleDownGapToWaterLines = buildGapToWaterLine(ctx.getStateFunc())
       //如果触发水位线中存在metric的实时用量无法获取，则获取具有最高ActionPriority的一个throttleAble的metric对所选择的所有pod进行压制操作
       if ThrottoleDownGapToWaterLines.HasUsageMissedMetric() {
           highestPrioriyMetric := ThrottleDownWaterLine.GetHighestPriorityThrottleAbleMetric()
           errPodKeys = throttlePods(ctx, &totalReleased, highestPrioriyMetric)
       } else {
           var released ReleaseResource
           //遍历触发水位线的metric中可以量化的metric：如果metric具有排序方法则直接使用其SortFunc对pod进行排序，否则使用GeneralSorter排序；
           //之后使用其对应的操作方法对pod执行操作，并计算释放出来的对应metric的资源量，直到对应metric到水位线的差距已不存在
           for _, m := range metricsQuantified {
               if m.SortAble {
                   m.SortFunc(ThrottleDownPods)
               } else {
                   GeneralSorter(ThrottleDownPods)
               }
       
               for !ThrottoleDownGapToWaterLines.TargetGapsRemoved(m) {
                   for index, _ := range ThrottleDownPods {
                       released = m.ThrottleFunc(ctx, index, ThrottleDownPods, &totalReleased)
                       ThrottoleDownGapToWaterLines[m] -= released[m]
                   }
               }
           }
       }
   }
   ```
关于扩展自定义指标和排序参考 "自定义指标干扰检测回避和自定义排序" 部分

## 增强的旁路cpuset管理能力
kubelet支持static的cpu manager策略，当guaranteed pod运行在节点上时，kebelet会为该pod分配指定的专属cpu，其他进程无法占用，这保证了guaranteed pod的cpu独占，但是也造成了cpu和节点的的利用率较低，造成了一定的浪费。
crane agent为cpuset管理提供了新的策略，允许pod和其他pod共享cpu当其指定了cpu绑核时，可以在利用绑核更少的上下文切换和更高的缓存亲和性的优点的前提下，还能让其他workload部署共用，提升资源利用率。
  
1. 提供了3种pod cpuset类型：

- exclusive：绑核后其他container不能再使用该cpu，独占cpu
- share：绑核后其他container可以使用该cpu
- none：选择没有被exclusive pod的container占用的cpu，可以使用share类型的绑核

   share类型的绑核策略可以在利用绑核更少的上下文切换和更高的缓存亲和性的优点的前提下，还能让其他workload部署共用，提升资源利用率

2. 放宽了kubelet中绑核的限制

   原先需要所有container的CPU limit与CPU request相等 ，这里只需要任意container的CPU limit大于或等于1且等于CPU request即可为该container设置绑核


3. 支持在pod运行过程中修改pod的 cpuset policy，会立即生效 

    pod的cpu manager policy从none转换到share，从exclusive转换到share，均无需重启

使用方法：  
1. 设置kubelet的cpuset manager为"none"  
2. 通过pod annotation设置cpu manager policy

   `qos.gocrane.io/cpu-manager: none/exclusive/share`
   ```yaml
   apiVersion: v1
   kind: Pod
   metadata:
     annotations:
       qos.gocrane.io/cpu-manager: none/exclusive/share
   ```

## 预测算法增强的动态资源超卖
为了提高稳定性，通常用户在部署应用的时候会设置高于实际使用量的Request值，造成资源的浪费，为了提高节点的资源利用率，用户会搭配部署一些BestEffort的应用，利用闲置资源，实现超卖；
但是这些应用由于缺乏资源limit和request的约束和相关信息，调度器依旧可能将这些pod调度到负载较高的节点上去，这与我们的初衷是不符的，所以最好能依据节点的空闲资源量进行调度。

crane通过如下两种方式收集了节点的空闲资源量，综合后作为节点的空闲资源量，增强了资源评估的准确性：

1. 通过本地收集的cpu用量信息  
`nodeCpuCannotBeReclaimed := nodeCpuUsageTotal + exclusiveCPUIdle - extResContainerCpuUsageTotal`  

   exclusiveCPUIdle是指被cpu manager policy为exclusive的pod占用的cpu的空闲量，虽然这部分资源是空闲的，但是因为独占的原因，是无法被复用的，因此加上被算作已使用量

   extResContainerCpuUsageTotal是指被作为动态资源使用的cpu用量，需要减去以免被二次计算

2. 创建节点cpu使用量的TSP，默认情况下自动创建，会根据历史预测节点CPU用量
```yaml
apiVersion: v1
data:
  spec: |
    predictionMetrics:
    - algorithm:
        algorithmType: dsp
        dsp:
          estimators:
            fft:
            - highFrequencyThreshold: "0.05"
              lowAmplitudeThreshold: "1.0"
              marginFraction: "0.2"
              maxNumOfSpectrumItems: 20
              minNumOfSpectrumItems: 10
          historyLength: 3d
          sampleInterval: 60s
      resourceIdentifier: cpu
      type: ExpressionQuery
      expressionQuery:
        expression: 'sum(count(node_cpu_seconds_total{mode="idle",instance=~"({{.metadata.name}})(:\\d+)?"}) by (mode, cpu)) - sum(irate(node_cpu_seconds_total{mode="idle",instance=~"({{.metadata.name}})(:\\d+)?"}[5m]))'
    predictionWindowSeconds: 3600
kind: ConfigMap
metadata:
  name: noderesource-tsp-template
  namespace: default
```

结合预测算法和当前实际用量推算节点的剩余可用资源，并将其作为拓展资源赋予节点，pod可标明使用该扩展资源作为离线作业将空闲资源利用起来，以提升节点的资源利用率；

使用方法：  
部署pod时limit和request使用`gocrane.io/<$ResourceName>：<$value>`即可，如下
```yaml
spec: 
   containers:
   - image: nginx
     imagePullPolicy: Always
     name: extended-resource-demo-ctr
     resources:
       limits:
         gocrane.io/cpu: "2"
       requests:
         gocrane.io/cpu: "2"
```

## 弹性资源限制功能
原生的BestEffort应用缺乏资源用量的公平保证，Crane保证使用动态资源的BestEffort pod其cpu使用量被限制在其允许使用的合理范围内，agent保证使用扩展资源的pod实际用量也不会超过其声明限制，同时在cpu竞争时也能按照各自声明量公平竞争；同时使用弹性资源的pod也会受到水位线功能的管理。

使用方法：
部署pod时limit和request使用`gocrane.io/<$ResourceName>：<$value>`即可

## 自定义指标干扰检测回避和自定义排序
自定义指标干扰检测回避和自定义排序的使用同 精确执行回避动作 部分中介绍的流程，此处介绍如何自定义自己的指标参与干扰检测回避流程

为了更好的基于NodeQOSEnsurancePolicy配置的metric进行排序和精准控制，对metric引入属性的概念。

metric的属性包含如下几个，自定义的指标实现这些字段即可：

1. Name 表明了metric的名称，需要同collector模块中收集到的指标名称一致
2. ActionPriority 表示指标的优先级，0为最低，10为最高
3. SortAble 表明该指标是否可以排序，如果为true，需实现对应的SortFunc
4. SortFunc 对应的排序方法，排序方法可以排列组合一些通用方法，再结合指标自身的排序，将在下文详细介绍
5. ThrottleAble 表明针对该指标，是否可以对pod进行压制，例如针对cpu使用量这个metric，就有相对应的压制手段，但是对于memory使用量这种指标，就只能进行pod的驱逐，无法进行有效的压制
6. ThrottleQuantified 表明压制（restore）一个pod后，能否准确计算出经过压制后释放出的对应metric的资源量，我们将可以准确量化的指标称为可Quantified，否则为不可Quantified；
   比如cpu用量，可以通过限制cgroup用量进行压制，同时可以通过当前运行值和压制后的值计算压制后释放的cpu使用量；而比如memory usage就不属于压制可量化metric，因为memory没有对应的throttle实现，也就无法准确衡量压制一个pod后释放出来的memory资源具体用量；
7. ThrottleFunc，执行Throttle动作的具体方法，如果不可Throttle，返回的released为空
8. RestoreFunc，被Throttle后，执行恢复动作的具体方法，如果不可Restore，返回的released为空
9. EvictAble，EvictQuantified，EvictFunc 对evict动作的相关定义，具体内容和Throttle动作类似

```go
type metric struct {
	Name WaterLineMetric

	ActionPriority int

	SortAble bool
	SortFunc func(pods []podinfo.PodContext)

	ThrottleAble      bool
	ThrottleQuantified bool
	ThrottleFunc      func(ctx *ExecuteContext, index int, ThrottleDownPods ThrottlePods, totalReleasedResource *ReleaseResource) (errPodKeys []string, released ReleaseResource)
	RestoreFunc       func(ctx *ExecuteContext, index int, ThrottleUpPods ThrottlePods, totalReleasedResource *ReleaseResource) (errPodKeys []string, released ReleaseResource)

	EvictAble      bool
	EvictQuantified bool
	EvictFunc      func(wg *sync.WaitGroup, ctx *ExecuteContext, index int, totalReleasedResource *ReleaseResource, EvictPods EvictPods) (errPodKeys []string, released ReleaseResource)
}
```

用户可以自行定义自己的metric，在构造完成后，通过registerMetricMap()进行注册

针对需要自定义的指标，可以通过实现如下的方法，搭配通用的排序方法即可方便地实现pod的灵活自定义排序，以代表自定义metric指标，<metric-sort-func>代表自定义的针对的排序策略
```yaml
func <metric>Sorter(pods []podinfo.PodContext) {
  orderedBy(classAndPriority, <metric-sort-func>, runningTime).Sort(pods)
}
```
其中`<metric-sort-func>`需要实现如下的排序方法
`func (p1, p2 podinfo.PodContext) int32` 