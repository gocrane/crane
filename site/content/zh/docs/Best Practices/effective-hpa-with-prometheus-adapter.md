---
title: "基于 Effective HPA 实现自定义指标的智能弹性实践"
weight: 10
description: >
  Effective HPA 的最佳实践.
---

Kubernetes HPA 支持了丰富的弹性扩展能力，Kubernetes 平台开发者部署服务实现自定义 Metric 的服务，Kubernetes 用户配置多项内置的资源指标或者自定义 Metric 指标实现自定义水平弹性。
Effective HPA 兼容社区的 Kubernetes HPA 的能力，提供了更智能的弹性策略，比如基于预测的弹性和基于 Cron 周期的弹性等。
Prometheus 是当下流行的开源监控系统，通过 Prometheus 可以获取到用户的自定义指标配置。

本文将通过一个例子介绍了如何基于 Effective HPA 实现自定义指标的智能弹性。部分配置来自于 [官方文档](https://github.com/kubernetes-sigs/prometheus-adapter/blob/master/docs/walkthrough.md)

## 部署环境要求

- Kubernetes 1.18+
- Helm 3.1.0
- Crane v0.6.0+
- Prometheus

参考 [安裝文档](https://docs.gocrane.io/dev/installation/) 在集群中安装 Crane，Prometheus 可以使用安装文档中的也可以是已部署的 Prometheus。

## 环境搭建

### 安装 PrometheusAdapter

Crane 组件 Metric-Adapter 和 PrometheusAdapter 都基于 [custom-metric-apiserver](https://github.com/kubernetes-sigs/custom-metrics-apiserver) 实现了 Custom Metric 和 External Metric 的 ApiService。在安装 Crane 时会将对应的 ApiService 安装为 Crane 的 Metric-Adapter，因此安装 PrometheusAdapter 前需要删除 ApiService 以确保 Helm 安装成功。

```bash
# 查看当前集群 ApiService
kubectl get apiservice 
```

因为安装了 Crane， 结果如下：

```bash
NAME                                   SERVICE                           AVAILABLE   AGE
v1beta1.batch                          Local                             True        35d
v1beta1.custom.metrics.k8s.io          Local                             True        18d
v1beta1.discovery.k8s.io               Local                             True        35d
v1beta1.events.k8s.io                  Local                             True        35d
v1beta1.external.metrics.k8s.io        crane-system/metric-adapter       True        18d
v1beta1.flowcontrol.apiserver.k8s.io   Local                             True        35d
v1beta1.metrics.k8s.io                 kube-system/metrics-service       True        35d
```

删除 crane 安装的 ApiService

```bash
kubectl delete apiservice v1beta1.external.metrics.k8s.io
```

通过 Helm 安装 PrometheusAdapter

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install prometheus-adapter -n crane-system prometheus-community/prometheus-adapter
```

再将 ApiService 改回 Crane 的 Metric-Adapter

```bash
kubectl apply -f https://raw.githubusercontent.com/gocrane/crane/main/deploy/metric-adapter/apiservice.yaml
```

### 配置 Metric-Adapter 开启 RemoteAdapter 功能

在安装 PrometheusAdapter 时没有将 ApiService 指向 PrometheusAdapter，因此为了让 PrometheusAdapter 也可以提供自定义 Metric，通过 Crane Metric Adapter 的 `RemoteAdapter` 功能将请求转发给 PrometheusAdapter。

修改 Metric-Adapter 的配置，将 PrometheusAdapter 的 Service 配置成 Crane Metric Adapter 的 RemoteAdapter

```bash
# 查看当前集群 ApiService
kubectl edit deploy metric-adapter -n crane-system
```

根据 PrometheusAdapter 的配置做以下修改：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: metric-adapter
  namespace: crane-system
spec:
  template:
    spec:
      containers:
      - args:
          #添加外部 Adapter 配置
        - --remote-adapter=true
        - --remote-adapter-service-namespace=crane-system
        - --remote-adapter-service-name=prometheus-adapter
        - --remote-adapter-service-port=443
```

#### RemoteAdapter 能力

![remote adapter](/images/remote-adapter.png)

Kubernetes 限制一个 ApiService 只能配置一个后端服务，因此，为了在一个集群内使用 Crane 提供的 Metric 和 PrometheusAdapter 提供的 Metric，Crane 支持了 RemoteAdapter 解决此问题

- Crane Metric-Adapter 支持配置一个 Kubernetes Service 作为一个远程 Adapter
- Crane Metric-Adapter 处理请求时会先检查是否是 Crane 提供的 Local Metric，如果不是，则转发给远程 Adapter

## 运行例子

### 准备应用

将以下应用部署到集群中，应用暴露了 Metric 展示每秒收到的 http 请求数量。

<summary>sample-app.deploy.yaml</summary>

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sample-app
  labels:
    app: sample-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: sample-app
  template:
    metadata:
      labels:
        app: sample-app
    spec:
      containers:
      - image: luxas/autoscale-demo:v0.1.2
        name: metrics-provider
        resources:
          limits:
            cpu: 500m
          requests:
            cpu: 200m
        ports:
        - name: http
          containerPort: 8080
```

<summary>sample-app.service.yaml</summary>

```yaml
apiVersion: v1
kind: Service
metadata:
  labels:
    app: sample-app
  name: sample-app
spec:
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: 8080
  selector:
    app: sample-app
  type: ClusterIP
```

```bash
kubectl create -f sample-app.deploy.yaml
kubectl create -f sample-app.service.yaml
```

当应用部署完成后，您可以通过命令检查 `http_requests_total` Metric：

```bash
curl http://$(kubectl get service sample-app -o jsonpath='{ .spec.clusterIP }')/metrics
```

### 配置采集规则

配置 Prometheus 的 ScrapeConfig，收集应用的 Metric: http_requests_total

```bash
kubectl edit configmap -n crane-system prometheus-server
```

添加以下配置

```yaml
    - job_name: sample-app
      kubernetes_sd_configs:
      - role: pod
      relabel_configs:
      - action: keep
        regex: default;sample-app-(.+)
        source_labels:
        - __meta_kubernetes_namespace
        - __meta_kubernetes_pod_name
      - action: labelmap
        regex: __meta_kubernetes_pod_label_(.+)
      - action: replace
        source_labels:
        - __meta_kubernetes_namespace
        target_label: namespace
      - source_labels: [__meta_kubernetes_pod_name]
        action: replace
        target_label: pod
```

此时，您可以在 Prometheus 查询 psql：sum(rate(http_requests_total[5m])) by (pod)

### 验证 PrometheusAdapter

PrometheusAdapter 默认的 Rule 配置支持将 http_requests_total 转换成 Pods 类型的 Custom Metric，通过命令验证：

```bash
kubectl get --raw /apis/custom.metrics.k8s.io/v1beta1 | jq . 
```

结果应包括 `pods/http_requests`:

```bash
{
  "name": "pods/http_requests",
  "singularName": "",
  "namespaced": true,
  "kind": "MetricValueList",
  "verbs": [
    "get"
  ]
}
```

这表明现在可以通过 Pod Metric 配置 HPA。

### 配置弹性

现在我们可以创建 Effective HPA。此时 Effective HPA 可以通过 Pod Metric `http_requests` 进行弹性：

#### 如何定义一个自定义指标开启预测功能

在 Effective HPA 的 Annotation 按以下规则添加配置：

```yaml
annotations:
  # metric-query.autoscaling.crane.io 是固定的前缀，后面是 Metric 的 type 和 名字，需跟 spec.metrics 中的 Metric.name 相同，支持 Pods 类型(pods)和 External 类型(external)
  metric-query.autoscaling.crane.io/pods.http_requests: "sum(rate(http_requests_total[5m])) by (pod)"
```

<summary>sample-app-hpa.yaml</summary>

```yaml
apiVersion: autoscaling.crane.io/v1alpha1
kind: EffectiveHorizontalPodAutoscaler
metadata:
  name: php-apache
  annotations:
    # metric-query.autoscaling.crane.io 是固定的前缀，后面是 Metric 的 type 和 名字，需跟 spec.metrics 中的 Metric.name 相同，支持 Pods 类型(pods)和 External 类型(external)
    metric-query.autoscaling.crane.io/pods.http_requests: "sum(rate(http_requests_total[5m])) by (pod)"
spec:
  # ScaleTargetRef is the reference to the workload that should be scaled.
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: sample-app
  minReplicas: 1        # MinReplicas is the lower limit replicas to the scale target which the autoscaler can scale down to.
  maxReplicas: 10       # MaxReplicas is the upper limit replicas to the scale target which the autoscaler can scale up to.
  scaleStrategy: Auto   # ScaleStrategy indicate the strategy to scaling target, value can be "Auto" and "Manual".
  # Metrics contains the specifications for which to use to calculate the desired replica count.
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 50
  - type: Pods
    pods:
      metric:
        name: http_requests
      target:
        type: AverageValue
        averageValue: 500m
  # Prediction defines configurations for predict resources.
  # If unspecified, defaults don't enable prediction.
  prediction:
    predictionWindowSeconds: 3600   # PredictionWindowSeconds is the time window to predict metrics in the future.
    predictionAlgorithm:
      algorithmType: dsp
      dsp:
        sampleInterval: "60s"
        historyLength: "7d"
```

```bash
kubectl create -f sample-app-hpa.yaml
```

查看 TimeSeriesPrediction 状态，如果应用运行时间较短，可能会无法预测：

```yaml
apiVersion: prediction.crane.io/v1alpha1
kind: TimeSeriesPrediction
metadata:
  creationTimestamp: "2022-07-11T16:10:09Z"
  generation: 1
  labels:
    app.kubernetes.io/managed-by: effective-hpa-controller
    app.kubernetes.io/name: ehpa-php-apache
    app.kubernetes.io/part-of: php-apache
    autoscaling.crane.io/effective-hpa-uid: 1322c5ac-a1c6-4c71-98d6-e85d07b22da0
  name: ehpa-php-apache
  namespace: default
spec:
  predictionMetrics:
    - algorithm:
        algorithmType: dsp
        dsp:
          estimators: {}
          historyLength: 7d
          sampleInterval: 60s
      resourceIdentifier: crane_pod_cpu_usage
      resourceQuery: cpu
      type: ResourceQuery
    - algorithm:
        algorithmType: dsp
        dsp:
          estimators: {}
          historyLength: 7d
          sampleInterval: 60s
      expressionQuery:
        expression: sum(rate(http_requests_total[5m])) by (pod)
      resourceIdentifier: pods.http_requests
      type: ExpressionQuery
  predictionWindowSeconds: 3600
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: sample-app
    namespace: default
status:
  conditions:
    - lastTransitionTime: "2022-07-12T06:54:42Z"
      message: not all metric predicted
      reason: PredictPartial
      status: "False"
      type: Ready
  predictionMetrics:
    - ready: false
      resourceIdentifier: crane_pod_cpu_usage
    - prediction:
        - labels:
            - name: pod
              value: sample-app-7cfb596f98-8h5vv
          samples:
            - timestamp: 1657608900
              value: "0.01683"
            - timestamp: 1657608960
              value: "0.01683"
            ......
      ready: true
      resourceIdentifier: pods.http_requests  
```

查看 Effective HPA 创建的 HPA 对象，可以观测到已经创建出基于自定义指标预测的 Metric: `crane_custom.pods_http_requests`

```yaml
apiVersion: autoscaling/v2beta2
kind: HorizontalPodAutoscaler
metadata:
  creationTimestamp: "2022-07-11T16:10:10Z"
  labels:
    app.kubernetes.io/managed-by: effective-hpa-controller
    app.kubernetes.io/name: ehpa-php-apache
    app.kubernetes.io/part-of: php-apache
    autoscaling.crane.io/effective-hpa-uid: 1322c5ac-a1c6-4c71-98d6-e85d07b22da0
  name: ehpa-php-apache
  namespace: default
spec:
  maxReplicas: 10
  metrics:
  - pods:
      metric:
        name: http_requests
      target:
        averageValue: 500m
        type: AverageValue
    type: Pods
  - pods:
      metric:
        name: pods.http_requests
        selector:
          matchLabels:
            autoscaling.crane.io/effective-hpa-uid: 1322c5ac-a1c6-4c71-98d6-e85d07b22da0
      target:
        averageValue: 500m
        type: AverageValue
    type: Pods
  - resource:
      name: cpu
      target:
        averageUtilization: 50
        type: Utilization
    type: Resource
  minReplicas: 1
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: sample-app
```

## 总结

由于生产环境的复杂性，基于多指标的弹性（CPU/Memory/自定义指标）往往是生产应用的常见选择，因此 Effective HPA 通过预测算法覆盖了多指标的弹性，达到了帮助更多业务在生产环境落地水平弹性的成效。
