
---
title: "Crane v0.7：通过控制台一键节省云成本"
linkTitle: "Release v0.7"
---
Crane（ Cloud Resource Analytics and Economics ） 是一个依托FinOps理论指导，基于云原生技术栈的云资源分析与成本优化平台。它的愿景是在保证客户应用运行质量的前提下实现极致的降本。

近期，Crane 发布了 0.7.0 版本。在新版本里我们提供了大量的新功能和优化，包括智能推荐框架 Recommendation Framework 以及全新改版的 Crane 产品化控制台。

详细的 Release Note 请见：https://github.com/gocrane/crane/releases/tag/v0.7.0

## 资源推荐框架 Recommendation Framework

Crane 的资源推荐，副本推荐功能在腾讯内部落地帮助自研业务每月节省了大量的成本，取得了很好的效果，详情请见：[https://mp.weixin.qq.com/s/1SeMzcf_VRvRysZ9NLI-Sw](https://mp.weixin.qq.com/s/1SeMzcf_VRvRysZ9NLI-Sw) 。同时，我们认为自动分析集群资源找到浪费并给出优化建议是帮助企业降本的重要方法，引入更多的分析类型至关重要。

因此在 0.7.0 版本中，Crane 设计了 Recommendation Framework，它提供了一个可扩展的推荐框架以支持多种云资源的分析，并内置了多种推荐器：资源推荐，副本推荐，闲置资源推荐。Recommendation Framework 通过 `RecommendationRule` 和 `Recommendation` CRD 描述了如何进行资源的分析推荐。

智能推荐的规则
```yaml
apiVersion: analysis.crane.io/v1alpha1
kind: RecommendationRule
metadata:
  name: workloads-rule
  labels:
    analysis.crane.io/recommendation-rule-preinstall: "true"
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

推荐的结果
```yaml
kind: Recommendation
apiVersion: analysis.crane.io/v1alpha1
metadata:
  name: workloads-rule-resource-flzbv
  namespace: crane-system
  labels:
    analysis.crane.io/recommendation-rule-name: workloads-rule
    analysis.crane.io/recommendation-rule-recommender: Resource
    analysis.crane.io/recommendation-rule-uid: 18588495-f325-4873-b45a-7acfe9f1ba94
    app: craned
    app.kubernetes.io/instance: crane
    app.kubernetes.io/managed-by: Helm
    app.kubernetes.io/name: crane
    app.kubernetes.io/version: v0.7.0
    helm.sh/chart: crane-0.7.0
  ownerReferences:
    - apiVersion: analysis.crane.io/v1alpha1
      kind: RecommendationRule
      name: workloads-rule
      uid: 18588495-f325-4873-b45a-7acfe9f1ba94
      controller: false
      blockOwnerDeletion: false
  spec:
    targetRef:                 # 推荐目标资源
      kind: Deployment
      namespace: crane-system
      name: craned
      apiVersion: apps/v1
    type: Resource
    completionStrategy:
      completionStrategyType: Once
    adoptionType: StatusAndAnnotation 
  status:
    recommendedInfo: >-
    {"spec":{"template":{"spec":{"containers":[{"name":"craned","resources":{"requests":{"cpu":"229m","memory":"120586239"}}},{"name":"dashboard","resources":{"requests":{"cpu":"114m","memory":"120586239"}}}]}}}}            # 推荐配置，可通过 kubectl patch 更新 deployment 配置
  currentInfo: >-
    {"spec":{"template":{"spec":{"containers":[{"name":"craned","resources":{"requests":{"cpu":"0","memory":"0"}}},{"name":"dashboard","resources":{"requests":{"cpu":"0","memory":"0"}}}]}}}}                       # 当前配置，可通过 kubectl patch 回滚已经采纳建议的 deployment 配置
    action: Patch                                     # 建议的 Action
```

目前 Crane 支持了以下推荐能力：

- **资源推荐**: 通过 VPA 算法分析应用的真实用量推荐更合适的资源配置
- **副本数推荐**: 通过 HPA 算法分析应用的真实用量推荐更合适的副本数量
- **闲置节点推荐**: 扫描集群中的空闲节点
- **闲置PV推荐**: 扫描集群中的未挂载 Pod 的 PV

用户可以基于 Recommendation Framework 通过实现 Recommender 接口**快速开发新的推荐能力或者扩展现有的推荐能力**。欢迎大家一起共建，丰富各类云资源的推荐能力，比如：GPU 的资源推荐，腾讯云 CLB 的闲置资源推荐。

## Crane Dashboard

Crane 在0.7.0 版本中花了大量时间基于 TDesign 的前端模版重构了 Dashboard，提供了一套美观、实用的产品界面。

新手用户可以观看以下视频对 Dashboard 有一个初步的了解：
<iframe src="https://user-images.githubusercontent.com/35299017/186680122-d7756b47-06be-44cb-8553-1957eaa3ed45.mp4"
scrolling="no" border="0" frameborder="no" framespacing="0" allowfullscreen="true" width="1000" height="600"></iframe>

视频中将一次成本优化过程分成了三个步骤：

1. **成本展示**: Kubernetes 资源( Deployments, StatefulSets )的多维度聚合与展示。
2. **成本分析**: 周期性的分析集群资源的状态并提供优化建议。
3. **成本优化**: 通过丰富的优化工具更新配置达成降本的目标。

通过 Dashboard，用户可以了解如何通过 Dashboard 快速上手，开启成本优化之旅。

我们也提供了 Crane Dashboard 的 **Live Demo**，欢迎体验：http://dashboard.gocrane.io/

### 成本分析

用户可以在 Dashboard 的 菜单->成本分析->资源推荐 中使用各种推荐能力，通过推荐功能，用户可以发现集群中的资源浪费，页面上的“采纳建议” 会提供一条 kubectl 命令，用户可以在他的环境中执行命令更新资源配置实现降本，后续会提供一键采纳建议的功能。

资源推荐：
![image](https://qcloudimg.tencent-cloud.cn/raw/cb640bcaf1ceaab768c321e0763d62bf.png)

副本数推荐：
![image](https://qcloudimg.tencent-cloud.cn/raw/613e5b342a96e044710026a3d991ab48.png)

### 碳排放计算器

用户可以在 Dashboard 的 菜单->成本洞察->碳排放分析 中使用碳排放计算器：
![image](https://qcloudimg.tencent-cloud.cn/raw/3413d877baae18586f76537b1f0e3d09.png)

详细的能力介绍请参考：[https://mp.weixin.qq.com/s/HQFTT7VG8M_q3WV7e4LRkg](https://mp.weixin.qq.com/s/HQFTT7VG8M_q3WV7e4LRkg)

## 升级指南

通过以下命令可以从旧版本的 Crane 升级到 0.7.0。

```yaml
kubectl apply -f https://raw.githubusercontent.com/gocrane/helm-charts/crane-0.7.0/charts/crane/crds/analysis.crane.io_analytics.yaml
kubectl apply -f https://raw.githubusercontent.com/gocrane/helm-charts/crane-0.7.0/charts/crane/crds/analysis.crane.io_recommendationrules.yaml
kubectl apply -f https://raw.githubusercontent.com/gocrane/helm-charts/crane-0.7.0/charts/crane/crds/analysis.crane.io_recommendations.yaml
kubectl apply -f https://raw.githubusercontent.com/gocrane/helm-charts/crane-0.7.0/charts/crane/crds/autoscaling.crane.io_effectivehorizontalpodautoscalers.yaml
kubectl apply -f https://raw.githubusercontent.com/gocrane/helm-charts/crane-0.7.0/charts/crane/crds/autoscaling.crane.io_effectiveverticalpodautoscalers.yaml

helm repo add crane https://gocrane.github.io/helm-charts
helm repo update
helm upgrade -n crane-system --install crane crane/crane --version 0.7.0
```

```
This is the final element on the page and there should be no margin below this.
```
