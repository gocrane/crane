## Dynamic resource oversold enhanced by prediction algorithm
In order to improve the stability, users usually set the request value higher than the actual usage when deploying applications, resulting in a waste of resources. In order to improve the resource utilization of nodes, users will deploy some besteffort applications in combination, using idle resources to realize oversold;
However, due to the lack of resource limit and request constraints and related information in these applications, scheduler may still schedule these pods to nodes with high load, which is inconsistent with our original intention, so it is best to schedule based on the free resources of nodes.

Crane collects the idle resources of nodes in the following two ways, and takes them as the idle resources of nodes after synthesis, which enhances the accuracy of resource evaluation:

Take cpu as an example, crane also supports the recovery of memory idle resources.

1. CPU usage information collected locally

`nodeCpuCannotBeReclaimed := nodeCpuUsageTotal + exclusiveCPUIdle - extResContainerCpuUsageTotal`

ExclusiveCPUIdle refers to the idle amount of CPU occupied by the pod whose CPU manager policy is exclusive. Although this part of resources is idle, it cannot be reused because of monopoly, so it is counted as used

ExtResContainerCpuUsageTotal refers to the CPU consumption used as dynamic resources, which needs to be subtracted to avoid secondary calculation

2. Create a TSP of node CPU usage, which is automatically created by default, and will predict node CPU usage based on history
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

Combine the prediction algorithm with the current actual consumption to calculate the remaining available resources of the node, and give it to the node as an extended resource. Pod can indicate that the extended resource is used as an offline job to use the idle resources, so as to improve the resource utilization rate of the node;

How to use:
When deploying pod, limit and request use `gocrane.io/<$resourcename>:<$value>`, as follows
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

## Elastic resource restriction function
The native besteffort application lacks a fair guarantee of resource usage. Crane guarantees that the CPU usage of the besteffort pod using dynamic resources is limited within the reasonable range of its allowable use. The agent guarantees that the actual consumption of the pod using extended resources will not exceed its stated limit. At the same time, when the CPU competes, it can also compete fairly according to its stated amount; At the same time, pod using elastic resources will also be managed by the watermark function.

How to use:
When deploying pod, limit and request use `gocrane.io/<$resourcename>:<$value>`

## suitable scene
In order to increase the load of nodes, some offline jobs or less important jobs can be scheduled and deployed to the cluster by using dynamic resources. Such jobs will use idle elastic resources.
With the watermark guarantee of QOS, when the node has a high load, it will be evicted and throttled first, and the utilization of the node will be improved on the premise of ensuring the stability of high-priority services.
See the section "Used with dynamic resources" in qos-interference-detection-and-active-avoidance.md.