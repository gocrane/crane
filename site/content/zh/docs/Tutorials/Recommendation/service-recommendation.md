---
title: "Service 推荐"
description: "Service 推荐功能介绍"
weight: 16
---

Service 推荐通过扫描集群中 Service 的运行状况，帮助用户找到闲置的 Kubernetes Service。

## 动机

通常在 Kubernetes 中我们会使用 Service + Workload 来自动创建和管理负载均衡并将负载均衡挂载到应用上，在日常的运营中难免会出现空闲和低利用率的负载均衡，浪费了大量成本，Service 推荐尝试帮助用户找到这部分 Service 来实现成本优化。

## 推荐示例

```yaml
apiVersion: analysis.crane.io/v1alpha1
kind: Recommendation
metadata:
  annotations:
    analysis.crane.io/last-start-time: "2023-06-12 11:52:23"
    analysis.crane.io/message: Success
    analysis.crane.io/run-number: "7823"
  creationTimestamp: "2023-06-12T09:44:23Z"
  labels:
    analysis.crane.io/recommendation-rule-name: service-rule
    analysis.crane.io/recommendation-rule-recommender: Service
    analysis.crane.io/recommendation-rule-uid: 67807cd9-b4c9-4d63-8493-d330ccace364
    analysis.crane.io/recommendation-target-kind: Service
    analysis.crane.io/recommendation-target-name: nginx
    analysis.crane.io/recommendation-target-namespace: crane-system
    analysis.crane.io/recommendation-target-version: v1
  name: service-rule-service-cnwt5
  namespace: crane-system
  ownerReferences:
    - apiVersion: analysis.crane.io/v1alpha1
      blockOwnerDeletion: false
      controller: false
      kind: RecommendationRule
      name: service-rule
      uid: 67807cd9-b4c9-4d63-8493-d330ccace364
spec:
  adoptionType: StatusAndAnnotation
  completionStrategy:
    completionStrategyType: Once
  targetRef:
    apiVersion: v1
    kind: Service
    name: nginx
    namespace: crane-system
  type: Service
status:
  action: Delete
  description: It is a Orphan Service, Pod count is 0
  lastUpdateTime: "2023-06-12T11:52:23Z"
```

在该示例中：

- 推荐的 TargetRef 指向了 Service：nginx
- 推荐类型为 Service 推荐
- action 是 Delete，这里只是给出建议

## 实现原理

Service 推荐按以下步骤完成一次推荐过程：

1. 扫描集群中所有 LoadBalancer 类型的 Service
2. 如果 Service 对应的 endpoints 中有 Address 或者 NotReadyAddresses，则不是限制的 Service
3. 依据 Service 推荐中流量相关 metric 检测 Service 是否小于阈值水位，如果小于水位则判定为闲置节点

## 如何验证推荐结果的准确性

以下是判断节点资源阈值水位的 Prom query，验证时把 node 替换成实际的节点名

```go
// Container network cumulative count of bytes received
queryFmtNetReceiveBytes = `sum(rate(container_network_receive_bytes_total{namespace="%s",pod=~"%s",container!=""}[3m]))`
// Container network cumulative count of bytes transmitted
queryFmtNetTransferBytes = `sum(rate(container_network_transmit_bytes_total{namespace="%s",pod=~"%s",container!=""}[3m]))`
```

## 支持的资源类型

只支持 Service 类型，目前只会对 LoadBalancer 类型的 Service 进行分析。

## 参数配置

| 配置项      | 默认值 | 描述                              |
|----------|-----|---------------------------------|
| net-receive-bytes | 0   | Service 对应 Pods 接受到的网络请求 bytes，默认不检查 |
| net-receive-percentile  | 0.99 | 计算接受到的网络请求时的 Percentile         |
| net-transfer-bytes | 0   | Service 对应 Pods 传输的网络请求 bytes，默认不检查   |
| net-transfer-percentile | 0.99    | 计算传输的网络请求时的 Percentile          |

注意，当 pod 配置了 liveness/readness probe 后，kubelet 的探测会带来一定的容器流量，因此流量的阈值需要设置的稍微大一些，可结合具体监控数据配置。

如何更新推荐的配置请参考：[**推荐框架**](/zh-cn/docs/tutorials/recommendation/recommendation-framework)