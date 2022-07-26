# EffectiveVerticalPodAutoscaler

EffectiveVerticalPodAutoscaler（简称 EVPA）是 Crane 提供的横向弹性伸缩产品，相比社区的 VPA 产品，EVPA 支持更丰富的弹性策略（预测，观测，周期），更灵活的扩展性。
 
- 丰富的弹性策略：支持分别配置扩容和缩容的弹性配置。
- 可扩展的弹性算法：支持用户自定义弹性算法，结合内置的 Effective VPA 算法支持各类弹性需求。
- 稳定性保障：提供了阈值控制、冷却时间的控制能力，提高了 Effective VPA 的稳定性。
- 可观测性：提供了一系列内置的观测指标，帮助用户更好的掌握集群中 Effective VPA 的状态。

## 产品功能

一个简单的 EVPA yaml 文件如下：

```yaml
apiVersion: autoscaling.crane.io/v1alpha1
kind: EffectiveVerticalPodAutoscaler
metadata:
  name: php-apache
spec:
  targetRef:
    apiVersion: "apps/v1"
    kind: Deployment
    name: php-apache
  resourcePolicy:
    containerPolicies:
      - containerName: 'php-apache'
        minAllowed:
          cpu: 100m
          memory: 50Mi
        maxAllowed:
          cpu: 1
          memory: 500Mi
        controlledResources: ["cpu", "memory"]
        scaleDownPolicy:
          metricThresholds:
            cpu:
              utilization: 35
            memory:
              utilization: 40
            mode: Auto
        stabilizationWindowSeconds: 43200
        scaleUpPolicy:
          metricThresholds:
            cpu:
              utilization: 95
            memory:
              utilization: 95
          mode: Auto
          stabilizationWindowSeconds: 150
```

1. ScaleTargetRef 配置你希望弹性的工作负载。
2. ScaleDownPolicy 定义了容器缩容策略。
3. ScaleUpPolicy 定义了容器扩容策略。

### ScaleDownPolicy 和 ScaleUpPolicy

#### 可扩展的弹性算法

#### 稳定性保障

#### 可观测性

### EffectiveVerticalPodAutoscaler status
Effective VPA 的 status 展示了推荐伸缩的资源值。

以下是一个 Effective VPA 的 Status yaml 例子：
```yaml
status:
  conditions:
    - lastTransitionTime: "2022-05-23T04:10:11Z"
      message: EffectiveVerticalPodAutoscaler is ready
      reason: EffectiveVerticalPodAutoscaler
      status: "True"
      type: Ready
  currentEstimators:
    - lastUpdateTime: "2022-05-23T04:10:11Z"
      recommendation:
        containerRecommendations:
          - containerName: php-apache
            target:
              cpu: 114m
              memory: "120586239"
      type: Percentile


```
