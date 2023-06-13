---
title: "闲置节点推荐"
description: "闲置节点推荐功能介绍"
weight: 15
---

闲置节点推荐通过扫描节点的状态和利用率，帮助用户找到闲置的 Kubernetes node。

## 动机

在使用 Kubernetes 的过程中，常常由于污点配置、label selector、低装箱率、低利用率等因素导致部分节点出现闲置状态，浪费了大量成本，闲置节点推荐尝试帮助用户找到这部分节点来实现成本优化。

## 推荐示例

```yaml
apiVersion: analysis.crane.io/v1alpha1
kind: Recommendation
metadata:
  annotations:
    analysis.crane.io/last-start-time: "2023-06-09 09:46:33"
    analysis.crane.io/message: Success
    analysis.crane.io/run-number: "111"
  creationTimestamp: "2023-05-31T11:06:10Z"
  generateName: idlenodes-rule-idlenode-
  generation: 111
  labels:
    analysis.crane.io/recommendation-rule-name: idlenodes-rule
    analysis.crane.io/recommendation-rule-recommender: IdleNode
    analysis.crane.io/recommendation-rule-uid: 25bf5a49-e78f-4f42-8e67-36c0b1b9bb5b
    analysis.crane.io/recommendation-target-kind: Node
    analysis.crane.io/recommendation-target-name: worker-node-1
    analysis.crane.io/recommendation-target-namespace: ""
    analysis.crane.io/recommendation-target-version: v1
  name: idlenodes-rule-idlenode-px2ck
  namespace: crane-system
  ownerReferences:
    - apiVersion: analysis.crane.io/v1alpha1
      blockOwnerDeletion: false
      controller: false
      kind: RecommendationRule
      name: idlenodes-rule
      uid: 25bf5a49-e78f-4f42-8e67-36c0b1b9bb5b
spec:
  adoptionType: StatusAndAnnotation
  completionStrategy:
    completionStrategyType: Once
  targetRef:
    apiVersion: v1
    kind: Node
    name: worker-node-1
  type: IdleNode
status:
  action: Delete
  description: Node is owned by DaemonSet
  lastUpdateTime: "2023-06-09T09:46:33Z"
```

在该示例中：

- 推荐的 TargetRef 指向了 Node：worker-node-1
- 推荐类型为闲置节点推荐
- action 是 Delete，但是下线节点是复杂操作，这里只是给出建议

## 实现原理

闲置节点推荐按以下步骤完成一次推荐过程：

1. 扫描集群中所有节点和节点上的 Pod
2. 如果节点上所有 Pod 都属于 DaemonSet，则判定为闲置节点
3. 依据 IdleNode 的其他配置检测节点是否小于阈值水位，如果小于水位则判定为闲置节点

## 如何验证推荐结果的准确性

以下是判断节点资源阈值水位的 Prom query，验证时把 node 替换成实际的节点名

```go
    // NodeCpuRequestUtilizationExprTemplate is used to query node cpu request utilization by promql, param is node name, node name which prometheus scrape
NodeCpuRequestUtilizationExprTemplate = `sum(kube_pod_container_resource_requests{node="%s", resource="cpu", unit="core"} * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)) by (node) / sum(kube_node_status_capacity{node="%s", resource="cpu", unit="core"} * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)) by (node) `
// NodeMemRequestUtilizationExprTemplate is used to query node memory request utilization by promql, param is node name, node name which prometheus scrape
NodeMemRequestUtilizationExprTemplate = `sum(kube_pod_container_resource_requests{node="%s", resource="memory", unit="byte", namespace!=""} * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)) by (node) / sum(kube_node_status_capacity{node="%s", resource="memory", unit="byte"} * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)) by (node) `
// NodeCpuUsageUtilizationExprTemplate is used to query node memory usage utilization by promql, param is node name, node name which prometheus scrape
NodeCpuUsageUtilizationExprTemplate = `sum(label_replace(irate(container_cpu_usage_seconds_total{instance="%s", container!="POD", container!="",image!=""}[1h]), "node", "$1", "instance",  "(^[^:]+)") * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)) by (node) / sum(kube_node_status_capacity{node="%s", resource="cpu", unit="core"} * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)) by (node) `
// NodeMemUsageUtilizationExprTemplate is used to query node memory usage utilization by promql, param is node name, node name which prometheus scrape
NodeMemUsageUtilizationExprTemplate = `sum(label_replace(container_memory_usage_bytes{instance="%s", namespace!="",container!="POD", container!="",image!=""}, "node", "$1", "instance", "(^[^:]+)") * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)) by (node) / sum(kube_node_status_capacity{node="%s", resource="memory", unit="byte"} * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)) by (node) `
```

## 支持的资源类型

只支持 Node，由于 Node 是 Cluster Scope 资源，因此 IdleNode 类型的 Recommendation 均在 crane-system namespace。

## 参数配置

| 配置项      | 默认值  | 描述                                       |
|----------|------|------------------------------------------|
| cpu-request-utilization | 0    | 高于该值利用率的节点不是闲置节点，0.5代表50%，默认不检查          |
| cpu-usage-utilization  | 0    | 高于该值 request 使用率的节点不是闲置节点，0.5代表50%，默认不检查 |
| cpu-percentile | 0.99 | 计算 cpu 负载时的 Percentile                   |
| memory-request-utilization | 0    | 高于该值利用率的节点不是闲置节点，0.5代表50%，默认不检查          |
| memory-usage-utilization  | 0    | 高于该值 request 使用率的节点不是闲置节点，0.5代表50%，默认不检查 |
| memory-percentile | 0.99 | 计算 memory 负载时的 Percentile                |

如何更新推荐的配置请参考：[**推荐框架**](/zh-cn/docs/tutorials/recommendation/recommendation-framework)