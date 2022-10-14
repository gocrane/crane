## 预测算法增强的动态资源超卖
为了提高稳定性，通常用户在部署应用的时候会设置高于实际使用量的Request值，造成资源的浪费，为了提高节点的资源利用率，用户会搭配部署一些BestEffort的应用，利用闲置资源，实现超卖；
但是这些应用由于缺乏资源limit和request的约束和相关信息，调度器依旧可能将这些pod调度到负载较高的节点上去，这与我们的初衷是不符的，所以最好能依据节点的空闲资源量进行调度。

crane通过如下两种方式收集了节点的空闲资源量，综合后作为节点的空闲资源量，增强了资源评估的准确性：

这里以cpu为例，同时也支持内存的空闲资源回收和计算。

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
         gocrane.io/memory: "2000Mi"
       requests:
         gocrane.io/cpu: "2"
         gocrane.io/memory: "2000Mi"
```

## 弹性资源限制功能
原生的BestEffort应用缺乏资源用量的公平保证，Crane保证使用动态资源的BestEffort pod其cpu使用量被限制在其允许使用的合理范围内，agent保证使用扩展资源的pod实际用量也不会超过其声明限制，同时在cpu竞争时也能按照各自声明量公平竞争；同时使用弹性资源的pod也会受到水位线功能的管理。

使用方法：
部署pod时limit和request使用`gocrane.io/<$ResourceName>：<$value>`即可

## 适配场景
为了提升节点的负载，可以将一些离线作业或者重要性较低的作业通过使用弹性资源的方式调度部署到集群中，这类作业会使用空闲的弹性资源，
搭配QOS的水位线保障，在节点出现负载较高的时候，也会优先被驱逐和压制，在保证高优先级业务稳定的前提下提升节点利用率。
可以参见qos-interference-detection-and-active-avoidance.zh.md中"与弹性资源搭配使用"部分的内容。