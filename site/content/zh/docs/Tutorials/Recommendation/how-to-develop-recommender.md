---
title: "如何开发 Recommender"
description: "介绍 Recommender 框架以及如何进行开发和扩展"
weight: 100
---

Recommendation Framework 提供了一套可扩展的 Recommender 框架并支持了内置的 Recommender，用户可以实现一个自定义的 Recommender，或者修改一个已有的 Recommender。

## Recommender Interface

```go
type Recommender interface {
	Name() string
	framework.Filter
	framework.PrePrepare
	framework.Prepare
	framework.PostPrepare
	framework.PreRecommend
	framework.Recommend
	framework.PostRecommend
	framework.Observe
}

// Phase: Filter

// Filter interface
type Filter interface {
    // The Filter will filter resource can`t be recommended via target recommender.
    Filter(ctx *RecommendationContext) error
}

// Phase: Prepare

// PrePrepare interface
type PrePrepare interface {
    CheckDataProviders(ctx *RecommendationContext) error
}

// Prepare interface
type Prepare interface {
    CollectData(ctx *RecommendationContext) error
}

type PostPrepare interface {
    PostProcessing(ctx *RecommendationContext) error
}

// PreRecommend interface
type PreRecommend interface {
    PreRecommend(ctx *RecommendationContext) error
}

// Phase: Recommend

// Recommend interface
type Recommend interface {
    Recommend(ctx *RecommendationContext) error
}

// PostRecommend interface
type PostRecommend interface {
    Policy(ctx *RecommendationContext) error
}

// Phase: Observe

// Observe interface
type Observe interface {
    Observe(ctx *RecommendationContext) error
}

```

Recommender 接口定义了一次推荐需要实现的四个阶段和八个扩展点。这些扩展点会在推荐过程中按顺序被调用。这些扩展点中的的一些可以改变推荐决策，而另一些仅用来提供信息。

## 架构

![](/images/recommendation-framework.png)

## 阶段

整个推荐过程分成了四个阶段：Filter，Prepare，Recommend，Observe。阶段的输入是需要分析的 Kubernetes 资源，输出是推荐的优化建议。 下面开始介绍每个阶段的输入、输出和能力。

`RecommendationContext` 保存了一次推荐过程中的上下文，包括推荐目标，RecommendationConfiguration 等，用户可以按需增加更多的内容。

### Filter

Filter 阶段用于预处理推荐数据。通常，在预处理时需判断推荐目标是否和 Recommender 匹配，比如，Resource Recommender 只支持处理 Workload（Deployment，StatefulSet）。除此之外，还可以判断推荐目标状态是否适合推荐，比如是否删除中，是否刚创建等。当返回 error 会终止此次推荐。BaseRecommender 实现了基本的预处理功能，用户可以调用它继承相关功能。

### Prepare

Prepare 阶段用于数据准备，请求外部监控系统并将时序数据保存在上下文中。PrePrepare 扩展点用于检测监控系统的链接情况。Prepare 扩展点用于查询时序数据。PostPrepare 扩展点用于对时序数据的数据处理，比如：应用冷启动的异常数据，部分数据的缺失，数据聚合，异常数据清理等。

### Recommend

Recommend 阶段用于基于时序数据和资源配置进行优化建议。优化建议的类型取决于推荐的类型。比如，如果是资源推荐，那么输出就是 kubernetes workload 的资源配置。Recommend 扩展点用于采用 Crane 的算法模块对数据进行分析计算，PostRecommend 阶段对分析结果进行最后处理。用户可以自定义 Recommend 阶段实现自定义的推荐结果。

### Observe

Observe 阶段用于推荐结果的可观测。比如，在资源推荐时，将优化建议的信息通过 Metric 保存到监控系统，再通过 Dashboard 观测优化建议带来的收益。
