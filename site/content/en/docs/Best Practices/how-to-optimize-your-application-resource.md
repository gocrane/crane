---
title: "How to optimize your application in FinOps era"
weight: 11
description: >
  How to optimize your application in FinOps era.
---

As more and more enterprises migrate their applications to the Kubernetes platform, it has gradually become an important entry point for resource orchestration and scheduling. As we all know, Kubernetes schedules applications based on the resource quotas requested by the applications, so how to properly configure application resource specifications has become the key to improving cluster utilization. This article will share how to correctly configure application resources based on the FinOps open-source project Crane, and how to promote resource optimization practices within the enterprise.

## Kubernetes How to manage resources

### Pod Resource model

In Kubernetes, the desired amount of resources for a Pod can be selectively set by specifying Request/Limit. When the resource Request is specified for a Container in a Pod, Kube-scheduler uses this information to determine which node to schedule the Pod on. When the resource Request and Limit are specified for a Container, kubelet ensures that the running container can access the requested resources through Cgroup parameters and does not use resources beyond the set limit. Kubelet also reserves system resources equal to the Request amount for the container to use.
example of resource configuration for a Pod：
```
apiVersion: v1
kind: Pod
metadata:
  name: frontend
spec:
  containers:
  - name: app
    image: images.my-company.example/app:v4
    resources:
      requests:
        memory: "64Mi"
        cpu: "250m"
      limits:
        memory: "128Mi"
        cpu: "500m"
```

Once the resource request amount is determined, the resource utilization formula for an application can be derived as follows: Utilization = Resource Usage / Resource Request.

Therefore, to improve the utilization of Pods, we need to configure reasonable resource requests.

### Workload Resource model

A workload is an application that runs on Kubernetes, consisting of a group of Pods, such as Deployments and StatefulSets. The number of Pods is referred to as the workload's replica count.

The resource utilization formula for a workload is: Workload Utilization = (Pod1 Usage + Pod2 Usage + ... PodN Usage) / (Request * Replicas).

As the formula shows, improving workload utilization can not only reduce the Request, but also reduce the Replicas.

### Common resource configuration issues

The Canadian software company Densify summarized common resource configuration issues in "12 RISK OF KUBERNETES RESOURCE MANAGEMENT" [1]. In the table below, we have added an analysis dimension of replica counts based on their findings.
|     | CPU Request                                                | Memory Request                                                  | CPU Limit                                           | Memory Limit                                      | Replicas                                  |
|-----|------------------------------------------------------------|-----------------------------------------------------------------|-----------------------------------------------------|---------------------------------------------------|-------------------------------------------|
| 过大  | 多余的CPU资源导致更多节点和资源的浪费	                                      | 调度器会申请过多Memory资源，导致更多节点和资源的浪费                                   | 允许Pod申请过多的CPU资源从而产生“吵闹邻居”风险，影响同一节点上的其他Pod           | 允许Pod申请过多的Memory资源从而产生“吵闹邻居”风险，从而影响同一节点上的其他Pod    | 多余的Pod会导致更多节点和资源的浪费                       |
| 过小  | 会导致在节点上过度堆叠Pod，如果所有CPU资源被用尽，则会在节点级别上产生争抢和CPU throttling的风险 | 	会导致在节点上过度堆叠Pod，如果所有Memory资源都被用尽，则会在节点级别上产生Pod终止的风险（OOM Killer） | 会限制Pod的CPU使用，如果实际业务压力超过Limit，会导致CPU throttling和性能下降 | 会限制Pod的Memory使用，如果实际业务压力超过Limit，会触发OOM Killer杀死进程 | 过少的Pod会带来过高的利用率，引发诸如性能下降，OOM Killer等稳定性问题 |
| 不设置 | 调度器将不确定在集群中可以调度多少Pod，并且过度堆叠的Pod会产生显著的性能风险和不均匀的负载           | 调度器将不确定在集群中可以调度多少Pod，从而产生过度堆叠和Pod被OOM Kill的风险                   | Pod将不受约束，放大“吵闹邻居”效应，并产生CPU throttling的风险            | Pod将不受约束，放大了“吵闹邻居”风险，如果节点内存耗尽，可能会导致OOM Killer启动   | N/A                                       |

As we can see, setting resource limits too low can lead to stability issues, while setting them too high only results in "mere" resource waste, which can be acceptable during periods of rapid business growth. This is the main reason why resource utilization rates are generally low for many businesses after migrating to the cloud. The following graph shows the resource usage of an application, with 30% resource waste between the peak historical usage of the Pod and its Request amount.
![Resource Waste](/images/resource-waste.jpg)

## Application Resource Optimization Model

After mastering Kubernetes' resource model, we can further derive a resource optimization model for cloud-native applications:

![Crane Overview](/images/resource-model.png)

The five lines in the graph from top to bottom are:

1. Node Capacity: The total amount of resources in all nodes in the cluster, corresponding to the Capacity of the cluster.
2. Allocated: The total amount of resources allocated by the application, corresponding to the Pod Request.
3. Weekly Peak: The peak resource usage of the application during a certain period in the past. Weekly peak can be used to predict future resource usage, and configuring resource specifications based on weekly peak has higher security and more general applicability.
4. Daily Average Peak: The peak resource usage of the application in the past day.
5. Mean: The average resource usage of the application, corresponding to Usage.

The idle resources can be divided into two categories:

1. Resource Slack: The difference between Capacity and Request.
2. Usage Slack: The difference between Request and Usage.

Total Slack = Resource Slack + Usage Slack

The goal of resource optimization is to reduce Resource Slack and Usage Slack. The model provides four steps for reducing waste, in order from top to bottom:

1. 提升装箱率：提升装箱率能够让 Capacity 和 Request 更加接近。手段有很多，例如：[动态调度器](/zh-cn/docs/tutorials/scheduling-pods-based-on-actual-node-load)、腾讯云原生节点的节点放大功能等
2. 业务规格调整减少资源锁定：根据周峰值资源用量调整业务规格使的 Request 可以减少到周峰值线。[资源推荐](/zh-cn/docs/tutorials/recommendation/resource-recommendation)和[副本推荐](/zh-cn/docs/tutorials/recommendation/replicas-recommendation)可以帮助应用实现此目标。
3. 业务规格调整+扩缩容兜底流量突发：在规格优化的基础上再通过 HPA 兜底突发流量使的 Request 可以减少到日均峰值线。此时 HPA 的目标利用率偏低，仅为应对突发流量，绝大多数时间内不发生自动弹性
4. 业务规格调整+扩缩容应对日常流量变化：在规格优化的基础上再通过 HPA 应用日常流量使的 Request 可以减少到均值。此时 HPA 的目标利用率等于应用的平均利用率

Based on this model, the open-source project Crane provides dynamic scheduling, recommendation framework, intelligent elasticity, and mixed deployment capabilities, realizing an all-in-one FinOps cloud resource optimization platform. In this article, we will focus on the recommendation framework.
## Optimizing resource configuration through the Crane recommendation framework

The open-source project Crane has launched the Recommendation Framework, which automatically analyzes the operation of various resources in the cluster and provides optimization suggestions. By analyzing CPU/Memory monitoring data over a period of time and using resource recommendation algorithms, the Recommendation Framework provides resource configuration suggestions, allowing enterprises to make decisions based on the proposed configurations.

In the following example, we will demonstrate how to quickly start a full cluster resource recommendation.

Before embarking on this cost-cutting journey, you need to install Crane in your environment. Please refer to Crane's installation documentation for guidance.

### 创建 RecommendationRule

下面是一个 RecommendationRule 示例： workload-rule.yaml。
```yaml
apiVersion: analysis.crane.io/v1alpha1
kind: RecommendationRule
metadata:
  name: workloads-rule
 spec:
  runInterval: 24h                            # 每24h运行一次
  resourceSelectors:                          # 资源的信息
    - kind: Deployment
      apiVersion: apps/v1
    - kind: StatefulSet
      apiVersion: apps/v1
  namespaceSelector:
    any: true                                 # 扫描所有namespace
  recommenders:                               # 使用 Workload 的副本和资源推荐器
    - name: Replicas
    - name: Resource
```

在该示例中： 
- 每隔24小时运行一次分析推荐，runInterval格式为时间间隔，比如: 1h，1m，设置为空表示只运行一次。
- 待分析的资源通过配置 resourceSelectors 数组设置，每个 resourceSelector 通过 kind，apiVersion，name 选择 k8s 中的资源，当不指定 name 时表示在 namespaceSelector 基础上的所有资源
- namespaceSelector 定义了待分析资源的 namespace，any: true 表示选择所有 namespace
- recommenders 定义了待分析的资源需要通过哪些 Recommender 进行分析。目前支持的类型：recommenders
- 资源类型和 recommenders 需要可以匹配，比如 Resource 推荐默认只支持 Deployments 和 StatefulSets，每种 Recommender 支持哪些资源类型请参考 recommender 的文档

1. 通过以下命令创建 RecommendationRule，刚创建时会立刻开始一次推荐。

```shell
kubectl apply -f workload-rules.yaml
```

这个例子会对所有 namespace 中的 Deployments 和 StatefulSets 做资源推荐和副本数推荐。
2. 检查 RecommendationRule 的推荐进度。通过 Status.recommendations 观察推荐任务的进度，推荐任务是顺序执行，如果所有任务的 lastStartTime 为最近时间且 message 有值，则表示这一次推荐完成

```shell
kubectl get rr workloads-rule
```

3. 通过以下命令查询推荐结果：

```shell
kubectl get recommend
```

可通过以下 label 筛选 Recommendation，比如 kubectl get recommend -l analysis.crane.io/recommendation-rule-name=workloads-rule

### 根据优化建议 Recommendation 调整资源配置
对于资源推荐和副本数推荐建议，用户可以 PATCH status.recommendedInfo 到 workload 更新资源配置，例如：

```shell
patchData=`kubectl get recommend workloads-rule-replicas-rckvb -n default -o jsonpath='{.status.recommendedInfo}'`;kubectl patch Deployment php-apache -n default --patch "${patchData}"
```

### Recommender

目前 Crane 支持了以下 Recommender：

- [**资源推荐**](/zh-cn/docs/tutorials/recommendation/resource-recommendation): 通过 VPA 算法分析应用的真实用量推荐更合适的资源配置
- [**副本数推荐**](/zh-cn/docs/tutorials/recommendation/replicas-recommendation): 通过 HPA 算法分析应用的真实用量推荐更合适的副本数量
- [**HPA 推荐**](/zh-cn/docs/tutorials/recommendation/hpa-recommendation): 扫描集群中的 Workload，针对适合适合水平弹性的 Workload 推荐 HPA 配置
- [**闲置节点推荐**](/zh-cn/docs/tutorials/recommendation/idlenode-recommendation): 扫描集群中的闲置节点

本文重点讨论 Workload 的资源配置优化，因此下面重点介绍资源推荐和副本推荐。

### 资源推荐

以下是一个资源推荐结果的样例：

```yaml
status:
  recommendedInfo: >-
        {"spec":{"template":{"spec":{"containers":[{"name":"craned","resources":{"requests":{"cpu":"150m","memory":"256Mi"}}},{"name":"dashboard","resources":{"requests":{"cpu":"150m","memory":"256Mi"}}}]}}}}
  currentInfo: >-
        {"spec":{"template":{"spec":{"containers":[{"name":"craned","resources":{"requests":{"cpu":"500m","memory":"512Mi"}}},{"name":"dashboard","resources":{"requests":{"cpu":"200m","memory":"256Mi"}}}]}}}}
  action: Patch
  conditions:
    - type: Ready
      status: 'True'
      lastTransitionTime: '2022-11-29T04:07:44Z'
      reason: RecommendationReady
      message: Recommendation is ready
  lastUpdateTime: '2022-11-30T03:07:49Z'
```

recommendedInfo 显示了推荐的资源配置，currentInfo 显示了当前的资源配置，格式是 Json ，可以通过 Kubectl Patch 将推荐结果更新到 TargetRef

#### 计算资源规格算法

The resource recommendation process is completed in the following steps:

1. Obtain the CPU and memory usage history of the workload in the past week through monitoring data.
2. Based on the historical usage, use the VPA Histogram to take the P99 percentile and multiply it by an amplification factor.
3. OOM Protection: If there have been historical OOM events in the container, consider increasing memory appropriately when making memory recommendations.
4. Resource Specification Regularization: Round up the recommended results to the specified container specifications.
The basic principle is to set the Request slightly higher than the maximum historical usage based on historical resource usage, and consider factors such as OOM and Pod specifications.

#### 副本推荐

以下是一个副本推荐结果的样例：

```yaml
status:
  recommendedInfo: '{"spec":{"replicas":1}}'
  currentInfo: '{"spec":{"replicas":2}}'
  action: Patch
  conditions:
    - type: Ready
      status: 'True'
      lastTransitionTime: '2022-11-28T08:07:36Z'
      reason: RecommendationReady
      message: Recommendation is ready
  lastUpdateTime: '2022-11-29T11:07:45Z'
```

The recommendedInfo displays the recommended replica count, and the currentInfo displays the current replica count in JSON format. The recommended results can be updated to TargetRef using Kubectl Patch.

The replica recommendation process is completed in the following steps:

1. Obtain the CPU and memory usage history of the workload in the past week through monitoring data.
2. Use the DSP algorithm to predict the future CPU usage for the next week.
3. Calculate the replica count for CPU and memory separately, and take the larger value.

#### 计算副本算法

以 CPU 举例，假设工作负载 CPU 历史用量的 P99 是10核，Pod CPU Request 是5核，目标峰值利用率是50%，可知副本数是4个可以满足峰值利用率不小于50%。

```yaml
replicas := int32(math.Ceil(workloadUsage / (TargetUtilization * float64(requestTotal) )))
```

### 和社区的差异

由资源优化模型可知，推荐框架能够将应用的 Request 降低到周峰值，并且推荐框架只做规格推荐，不执行变更，安全性更高、适用于更多业务类型。如果需要进一步降低 Request，可以考虑通过 HPA 等方案实现。

|            | 利用率                                    | 管理配置类型                    | 变更类型                                           |
|------------|----------------------------------------|---------------------------|------------------------------------------------|
| 社区 HPA	    | 平均利用率                                  | 副本数                       | 自动变更                                           |
| 社区 VPA     | 近似峰值利用率                                | 资源 Request                | 自动变更/建议                                        |
| Crane 推荐框架 | 周峰值利用率                                 | 副本数+资源 Request            | 自动变更/建议                                        |
| 推荐框架的优势    | 虽然周峰值利用率带来的降本空间较小，但是配置简单，更加安全，适用更多应用类型 | 可以同时推荐副本数+资源 Request，按需调整 | 提供CRD/Metric方式的推荐建议，方便集成用户的系统，未来支持通过CICD实现自动更新 |

## 最佳实践

FinOps 建议采用迭代方法来管理云服务的可变成本。持续管理的迭代由三个阶段组成：成本观测（Inform）、 成本分析（Recommend）和 成本优化（Operate）。下面我们将基于这三个阶段+腾讯内部的实践经验介绍如何使用 Crane 实现 K8S 资源的配置管理。

### 成本观测--计算成本/收益

成本观测是降本之旅的核心关键。只有明确了目标，降本优化才会有的放矢。因此，用户需要建立集群资源的监控观测系统，来评估是否需要进行降本增效。例如，集群的装箱率是多少？集群的平均/峰值利用率是多少？Namespace 的资源用量分布，Workload 的平均/峰值利用率是多少？

### 成本分析--建立系统
The Crane recommendation framework provides a complete set of analysis and optimization tools for full-fledged analysis of cluster resources, and records the recommended results in CRD and Metrics for easy integration into business systems.

The practice within Tencent is as follows:
1. Use RecommendationRule to recommend resources and replicas for all workloads in the cluster, updated every 12 hours.
2. Display the complete recommendation results separately in the control interface.
3. Display resource/replica recommendations on the workload data display page.
4. Display observation data of the workload in Grafana charts.
5. Provide OpenAPI for businesses to obtain recommendations and optimize them according to business needs.

### 成本优化--渐进式推进

The FinOps Foundation has defined a "crawl, walk, run" maturity method for FinOps, enabling enterprises to start small and gradually expand in scale, scope, and complexity. Similarly, the premise of cost reduction is to ensure stability, as changes in resource configuration and unreasonable configurations may affect business stability. User optimization processes should follow the same approach:

1. Verify the accuracy of the configuration in the CI/CD environment before updating the production environment.
2. Optimize businesses with severe waste first, and then optimize businesses with relatively low configurations.
3. Optimize non-core businesses first, and then optimize core businesses.
4. Configure recommended parameters based on business characteristics: Online businesses require more resource buffers, while offline businesses can accept higher utilization rates.
5. The release platform prompts users with recommended configurations and updates only after confirmation to prevent unexpected online changes.
6. Some business clusters automatically update workload configurations based on recommended suggestions to achieve higher utilization rates.

In the book "Cloud FinOps" which introduces FinOps, it shares an example of a Fortune 500 company optimizing resources through an automated system, with the following workflow:

![Resource flow](/images/resource-flow.png)

自动的配置优化在 FinOps 中属于高级阶段，推荐在实践 FinOps 的高级阶段中使用。不过至少，你应该考虑跟踪你的推荐，并且让对应的团队手动执行所需的变更。

## 展望未来

Whether or not resource optimization is needed, Crane can be used as a trial object when practicing FinOps. You can first understand the current state of the Kubernetes cluster through cost display, and choose the optimization method based on the problem. Resource configuration optimization, as introduced in this article, is the most direct and common method.

In the future, the Crane recommendation framework will evolve towards more accurate, intelligent, and rich goals:

-Integration with CI/CD frameworks: Automated configuration updates can further improve utilization rates compared to manual updates and are suitable for business scenarios with higher resource utilization rates.
-Cost left shift: Discover and solve resource waste earlier through configuration optimization in the CI/CD stage.
-Configuration recommendation based on application load characteristics: Identify load patterns and burst tasks based on algorithms and provide reasonable recommendations.
-Resource recommendation for task types: Currently, more support is provided for long-running online businesses, but resource recommendations can also optimize configuration for task-type applications.
-Analysis of more types of idle resources in Kubernetes: Scan idle resources in the cluster, such as Load Balancer/Storage/Node/GPU.

## Appendix
1.The Top 12 Kubernetes Resource Risks: K8s Best Practices: [Top 12 Kubernetes Resource Risks](https://www.densify.com/resources/k8s-resource-risks)


