---
title: "HPA 推荐（Alpha）"
description: "HPA 推荐介绍"
weight: 16
---

Kubernetes 用户希望使用 HPA 来实现按需使用，提示资源利用率。但是往往不知道哪些应用适合弹性也不知道如何配置HPA的参数。通过 HPA 推荐的算法分析应用的真实用量推荐合适的水平弹性的配置，您可以参考并采纳它提升应用资源利用率。

HPA 推荐还处于 Alpha 阶段，欢迎对功能提供意见。

## 动机

在 Kubernetes 中，HPA(HorizontalPodAutoscaler) 自动更新工作负载资源 （例如 Deployment 或者 StatefulSet）， 目的是自动扩缩工作负载以满足需求。但是在实际使用过程中我们观察到以下使用问题：

- 有些应用可以通过 HPA 提示资源利用率，但是没有配置 HPA
- 有些 HPA 配置并不合理，无法有效的进行弹性伸缩，也就达不到提示利用率的效果

HPA 推荐通过应用的历史数据结合算法分析给出建议：哪些应用适合配置 HPA 以及 HPA 的配置。

## 推荐示例

一个简单的弹性推荐 yaml 文件如下：

```yaml
apiVersion: analysis.crane.io/v1alpha1
kind: Recommendation
metadata:
  labels:
    analysis.crane.io/recommendation-rule-name: workload-hpa
    analysis.crane.io/recommendation-rule-recommender: HPA
    analysis.crane.io/recommendation-rule-uid: 0214c84b-8b39-499b-a7c6-559ac460695d
    analysis.crane.io/recommendation-target-kind: Rollout
    analysis.crane.io/recommendation-target-name: eshop
    analysis.crane.io/recommendation-target-version: v1alpha1
  name: workload-hpa-hpa-blr4r
  namespace: zytms
  ownerReferences:
    - apiVersion: analysis.crane.io/v1alpha1
      blockOwnerDeletion: false
      controller: false
      kind: RecommendationRule
      name: workload-hpa
      uid: 0214c84b-8b39-499b-a7c6-559ac460695d
spec:
  adoptionType: StatusAndAnnotation
  completionStrategy:
    completionStrategyType: Once
  targetRef:
    apiVersion: argoproj.io/v1alpha1
    kind: Rollout
    name: eshop
    namespace: eshop
  type: HPA
status:
  action: Create
  lastUpdateTime: "2022-12-05T06:12:54Z"
  recommendedInfo: '{"kind":"EffectiveHorizontalPodAutoscaler","apiVersion":"autoscaling.crane.io/v1alpha1","metadata":{"name":"eshop","namespace":"eshop","creationTimestamp":null},"spec":{"scaleTargetRef":{"kind":"Rollout","name":"eshop","apiVersion":"argoproj.io/v1alpha1"},"minReplicas":1,"maxReplicas":1,"scaleStrategy":"Preview","metrics":[{"type":"Resource","resource":{"name":"cpu","target":{"type":"Utilization","averageUtilization":58}}},{"type":"Pods","pods":{"metric":{"name":"k8s_pod_cpu_core_used"},"target":{"type":"AverageValue","averageValue":"500m"}}}]},"status":{}}'
  recommendedValue: |
    effectiveHPA:
      maxReplicas: 1
      metrics:
      - resource:
          name: cpu
          target:
            averageUtilization: 58
            type: Utilization
        type: Resource
      - pods:
          metric:
            name: k8s_pod_cpu_core_used
          target:
            averageValue: 500m
            type: AverageValue
        type: Pods
      minReplicas: 1
```

在该示例中：

- 推荐的 TargetRef 指向 eshop 的 Rollout：eshop
- 推荐类型为 HPA 推荐
- adoptionType 是 StatusAndAnnotation，表示将推荐结果展示在 recommendation.status 和 Deployment 的 Annotation
- recommendedInfo 显示了推荐的 EHPA 配置（recommendedValue 已经 deprecated）
- action 是 Create，如果集群中已经有 EHPA 存在，则 action 是 Patch

## 实现原理

HPA 推荐按以下步骤完成一次推荐过程：

1. 通过监控数据，获取 Workload 过去一周的 CPU 和 Memory 历史用量。
2. 用 DSP 算法预测未来一周 CPU 用量
3. 分别计算 CPU 和 内存分别对应的副本数，取较大值作为 minReplicas
4. 计算历史 CPU 用量的波动率和最小用量，筛选出适合使用 HPA 的 Workload
5. 根据 pod 的 CPU 峰值利用率计算 targetUtilization
6. 根据推荐的 targetUtilization 计算推荐的 maxReplicas
7. 将 targetUtilization，maxReplicas，minReplicas 组装成完整的 EHPA 对象作为推荐结果

### 如何筛选适合使用 HPA 的 workload

适合使用 HPA 的 Workload 需要满足以下条件：

1. Workload 运行基本正常，比如绝大多数 Pod 都处于运行中
2. CPU 的使用量存在波峰波谷的波动。如果基本没有波动或者完全随机的用量适合通过副本推荐配置固定的副本数
3. 有一定资源用量的 Workload，如果资源用量长期非常低，那么即使有一定的波动量，也是没有使用 HPA 的价值的

以下是一个典型的存在波峰波谷规律的 Workload 的历史资源用量

![](/images/algorithm/dsp/input0.png)

### 计算最小副本算法

方法和副本推荐中计算副本算法一致，请参考：[**副本推荐**](/zh-cn/docs/tutorials/recommendation/replicas-recommendation)

## 支持的资源类型

默认支持 StatefulSet 和 Deployment，但是支持所有实现了 Scale SubResource 的 Workload。

## 参数配置

| 配置项         | 默认值   | 描述                                   |
|-------------|-------|--------------------------------------|
| workload-min-replicas | 1     | 小于该值的工作负载不做弹性推荐                      |
| pod-min-ready-seconds | 30    | 定义了 Pod 是否 Ready 的秒数                 |
| pod-available-ratio | 0.5   | Ready Pod 比例小于该值的工作负载不做弹性推荐          |
| default-min-replicas | 1     | 最小 minReplicas                       |
| cpu-percentile | 0.95  | 历史 CPU 用量的 Percentile                |
| mem-percentile | 0.95  | 历史内存用量的 Percentile                   |
| cpu-target-utilization | 0.5   | CPU 目标峰值利用率                          |
| mem-target-utilization | 0.5   | 内存目标峰值利用率                            |
| predictable | false | 当设置成 true 时，如果 CPU 历史用量无法预测，则不进行推荐   |
| reference-hpa | true  | 推荐配置会参考现有 HPA 的配置，继承比如自定义指标等信息到 EHPA |
| min-cpu-usage-threshold | 1     | Workload CPU 最小用量，如果历史用量小于该配置，则不进行推荐 |
| fluctuation-threshold | 1.5   | Workload CPU 的波动率，小于该配置，则不进行推荐       |
| min-cpu-target-utilization | 30    | CPU 的 TargetUtilization 最小值          |
| max-cpu-target-utilization | 75    | CPU 的 TargetUtilization 最大值          |
| max-replicas-factor | 3     | 在计算 maxReplicas 时的放大系数               |

如何更新推荐的配置请参考：[**推荐框架**](/zh-cn/docs/tutorials/recommendation/recommendation-framework)