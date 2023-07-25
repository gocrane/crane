---
title: "推荐框架"
description: "介绍智能推荐框架的原理和用法"
weight: 10
---

**推荐框架**是指自动分析集群的各种资源的运行情况并给出优化建议。

## 推荐概览

Crane 的推荐模块定期的检测发现集群资源配置的问题，并给出优化建议。智能推荐提供了多种 Recommender 来实现面向不同资源的优化推荐。
如何你想了解 Crane 如何做智能推荐的，或者你想要尝试实现一个自定义的 Recommender，或者修改一个已有的 Recommender 的推荐规则，这篇文章将帮助你了解智能推荐。

## 用例

以下是智能推荐的典型用例：

- 创建 RecommendationRule 配置。RecommendationRule Controller 会根据配置定期运行推荐任务，给出优化建议 Recommendation。
- 根据优化建议 Recommendation 调整资源配置。

## 创建 RecommendationRule 配置

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

- 每隔24小时运行一次分析推荐，`runInterval`格式为时间间隔，比如: 1h，1m，设置为空表示只运行一次。
- 待分析的资源通过配置 `resourceSelectors` 数组设置，每个 `resourceSelector` 通过 kind，apiVersion，name 选择 k8s 中的资源，当不指定 name 时表示在 `namespaceSelector` 基础上的所有资源
- `namespaceSelector` 定义了待分析资源的 namespace，`any: true` 表示选择所有 namespace
- `recommenders` 定义了待分析的资源需要通过哪些 Recommender 进行分析。目前支持的类型：[recommenders](/zh-cn/docs/tutorials/recommendation/recommendation-framework#recommender)
- 资源类型和 `recommenders` 需要可以匹配，比如 Resource 推荐默认只支持 Deployments 和 StatefulSets，每种 Recommender 支持哪些资源类型请参考 recommender 的文档

1. 通过以下命令创建 RecommendationRule，刚创建时会立刻开始一次推荐。

```shell
kubectl apply -f workload-rules.yaml
```

这个例子会对所有 namespace 中的 Deployments 和 StatefulSets 做资源推荐和副本数推荐。

2. 检查 RecommendationRule 的推荐进度。通过 Recommendation 的 Annotation 可观察到任务的上次开始时间和运行的结果。

```shell
kubectl get recommend workloads-rule-replicas-7djlk -o yaml
```

```yaml
apiVersion: analysis.crane.io/v1alpha1
kind: Recommendation
metadata:
  annotations:
    analysis.crane.io/last-start-time: "2023-07-24 11:43:58"
    analysis.crane.io/message: 'Failed to run recommendation flow in recommender Replicas:
      Replicas CalculatePodTemplateRequests cpu failed: missing request for cpu'
    analysis.crane.io/run-number: "59"
  creationTimestamp: "2023-06-01T11:37:16Z"
```

3. 查看优化建议 `Recommendation`

可通过以下 label 筛选 `Recommendation`，比如 `kubectl get recommend -l analysis.crane.io/recommendation-rule-name=workloads-rule`

- RecommendationRule 名称：analysis.crane.io/recommendation-rule-name
- RecommendationRule UID：analysis.crane.io/recommendation-rule-uid
- RecommendationRule 的 recommender：analysis.crane.io/recommendation-rule-recommender
- 推荐资源的 kind：analysis.crane.io/recommendation-target-kind
- 推荐资源的 apiversion：analysis.crane.io/recommendation-target-apiversion
- 推荐资源的 name：analysis.crane.io/recommendation-target-apiversion

通常， `Recommendation` 的 namespace 等于推荐资源的 namespace。闲置节点推荐的 `Recommendation` 除外，它们在 Crane 的 root namespace 中，默认是 crane-system。

## 根据优化建议 Recommendation 调整资源配置

对于资源推荐和副本数推荐建议，用户可以 PATCH status.recommendedInfo 到 workload 更新资源配置，例如：

```shell
patchData=`kubectl get recommend workloads-rule-replicas-rckvb -n default -o jsonpath='{.status.recommendedInfo}'`;kubectl patch Deployment php-apache -n default --patch "${patchData}"
```

对于闲置节点推荐，由于节点的下线在不同平台上的步骤不同，用户可以根据自身需求进行节点的下线或者缩容。

## Recommender

目前 Crane 支持了以下 Recommender：

- [**资源推荐**](/zh-cn/docs/tutorials/recommendation/resource-recommendation): 通过 VPA 算法分析应用的真实用量推荐更合适的资源配置
- [**副本数推荐**](/zh-cn/docs/tutorials/recommendation/replicas-recommendation): 通过 HPA 算法分析应用的真实用量推荐更合适的副本数量
- [**HPA 推荐**](/zh-cn/docs/tutorials/recommendation/hpa-recommendation): 扫描集群中的 Workload，针对适合适合水平弹性的 Workload 推荐 HPA 配置
- [**闲置节点推荐**](/zh-cn/docs/tutorials/recommendation/idlenode-recommendation): 扫描集群中的闲置节点
- [**Service 推荐**](/zh-cn/docs/tutorials/recommendation/service-recommendation): 扫描集群中的闲置 Service
- [**PV 推荐**](/zh-cn/docs/tutorials/recommendation/pv-recommendation): 扫描集群中的闲置 PV

### Recommender 框架

Recommender 框架定义了一套工作流程，Recommender 按流程顺序执行，流程分为四个阶段：Filter,Prepare,Recommend,Observe，Recommender 通过实现这四个阶段完成推荐功能。

开发或者扩展 Recommender 请参考：[如何开发 Recommender](/zh-cn/docs/tutorials/recommendation/how-to-develop-recommender)

## RecommendationConfiguration

RecommendationConfiguration 定义了 recommender 的配置。部署时会在 crane root namespace创建一个 ConfigMap：recommendation-configuration，数据包括了一个 yaml 格式的 RecommendationConfiguration.

下面是一个 RecommendationConfiguration 示例。

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: recommendation-configuration
  namespace: crane-system
data:
  config.yaml: |-
    apiVersion: analysis.crane.io/v1alpha1
    kind: RecommendationConfiguration
    recommenders:
      - name: Replicas
        acceptedResources:                   # 接受的资源类型
          - kind: Deployment
            apiVersion: apps/v1
          - kind: StatefulSet
            apiVersion: apps/v1
        config:                              # 设置 recommender 的参数
          workload-min-replicas: "1"         
      - name: Resource
        acceptedResources:                   # 接受的资源类型
          - kind: Deployment
            apiVersion: apps/v1
          - kind: StatefulSet
            apiVersion: apps/v1
```

用户可以修改 ConfigMap 内容并重新发布 Crane，触发新的配置生效。

## 如何让推荐结果更准确

应用在监控系统（比如 Prometheus）中的历史数据越久，推荐结果就越准确，建议生产上超过两周时间。对新建应用的预测往往不准。
