---
title: "定义自己的水位线指标"
description: "如何自定义水位线指标"
weight: 22
---

## 自定义指标干扰检测回避和自定义排序
自定义指标干扰检测回避和自定义排序的使用同 精确执行回避动作 部分中介绍的流程，此处介绍如何自定义自己的指标参与干扰检测回避流程

为了更好的基于NodeQOS配置的metric进行排序和精准控制，对metric引入属性的概念。

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