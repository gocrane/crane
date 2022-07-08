# 基于 Effective HPA 实现自定义指标的智能弹性实践

Kubernetes HPA 支持了丰富的弹性扩展能力，Kubernetes 平台开发者部署服务实现自定义 Metric 的服务，Kubernetes 用户配置多项内置的资源指标或者自定义 Metric 指标实现自定义水平弹性。
Crane Effective HPA 兼容社区的 Kubernetes HPA 的能力，提供了更智能的弹性策略，比如基于预测的弹性和基于 Cron 周期的弹性等。
Prometheus 是当下流行的开源监控系统，通过 Prometheus 可以获取到用户的自定义指标配置。

本文将通过一个例子介绍了如何基于 Effective HPA 实现自定义指标的智能弹性。

## 部署环境要求

- Kubernetes 1.18+
- Helm 3.1.0
- Crane v0.6.0+

## 环境搭建

### 集群版本

版本：v1.22.9
注：为了接入阿里云ECI以降低成本，需要指定节点进行副本扩缩，故采用该版本作为基础环境

```bash
# kubectl version

Client Version: version.Info{Major:"1", Minor:"22", GitVersion:"v1.22.9", GitCommit:"6df4433e288edc9c40c2e344eb336f63fad45cd2", GitTreeState:"clean", BuildDate:"2022-04-13T19:57:43Z", GoVersion:"go1.16.15", Compiler:"gc", Platform:"linux/amd64"}
Server Version: version.Info{Major:"1", Minor:"22", GitVersion:"v1.22.9", GitCommit:"6df4433e288edc9c40c2e344eb336f63fad45cd2", GitTreeState:"clean", BuildDate:"2022-04-13T19:52:02Z", GoVersion:"go1.16.15", Compiler:"gc", Platform:"linux/amd64"}
```

### Prometheus

镜像及版本：registry.cn-hangzhou.aliyuncs.com/istios/prometheus-adapter-amd64:v0.9.0
注：EHPA需要兼容目前已有的外部指标，该部分通过prometheus-adapter实现

#### 指标采集

服务设置

```bash
# kubectl -n cloud get deployment metric-source-service -o yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: metric-source-service
  namespace: cloud
spec:
  template:
    metadata:
      annotations:
        prometheus.aispeech.com/metric_path: /metrics
        prometheus.aispeech.com/scrape_port: "28002"
        prometheus.aispeech.com/scrape_scheme: http
        prometheus/should_be_scraped: "true"
```

接口验证

```bash
# kubectl -n cloud get pods -owide
NAME READY STATUS RESTARTS AGE
metric-source-service-6c6b4b4648-n7bmc 1/1 Running 0 14d
# curl 10.244.0.59:28002/metrics
mock_traffic{} 13820
```

#### 外部指标创建

注：指标来源与需要应用该指标实现扩缩容的服务跨namespace，设置为false

```bash
# kubectl -n ${prometheus-adapter-namespace} get configmap prometheus-adapter-config -o yaml
apiVersion: v1
data:
  config.yaml: |
    externalRules:
    - seriesQuery:'{__name__="mock_traffic",pod_name!=""}'
      metricsQuery: max(<<.Series>>{<<.LabelMatchers>>}) by (pod_name)
      resources:
        namespaced: false

```

配置添加后重启prometheus-adapter
查询外部指标状态

external-apiservice

```bash
# kubectl get apiservice v1beta1.external.metrics.k8s.io
NAME SERVICE AVAILABLE AGE
v1beta1.external.metrics.k8s.io monitoring/prometheus-adapter True 36d
# kubectl get --raw /apis/external.metrics.k8s.io/v1beta1|tee |python -m json.tool
{
  "apiVersion": "v1",
  "groupVersion": "external.metrics.k8s.io/v1beta1",
  "kind":"APIResourceList",
  "resources": [
    {
        "kind": "ExternalMetricValueList",
        "name": "mock_traffic",
        "namespaced": true,
        "singularName":"",
        "verbs": [
          "get"
        ]
      }
    ]
}

```

### Metric-Adapter

镜像及版本：docker.io/gocrane/metric-adapter:v0.5.0-tke.1-7-g10ddeb6
注：时序预测模型功能需要通过metric-adapter获取，因此需要对prometheus-adapter与metric-adapter进行指标集成

#### 配置remote-adapter
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
#添加外部Adapter
        - --remote-adapter=true
        - --remote-adapter-service-namespace=monitoring
        - --remote-adapter-service-name=prometheus-adapter
        - --remote-adapter-service-port=443
```

#### 修改apiservice

指定外部指标源为metric-adapter，prometheus-adapter指标通过metric-adapter代理

```bash
# kubectl edit apiservice v1beta1.external.metrics.k8s.io
# kubectl get apiservice v1beta1.external.metrics.k8s.io -o yaml

#外部指标正常
# kubectl get --raw /apis/external.metrics.k8s.io/v1beta1|tee |python -m json.tool
{
  "apiVersion":"v1",
  "groupVersion":"external.metrics.k8s.io/v1beta1",
  "kind":"APIResourceList",
  "resources": [
  {
    "kind":"ExternalMetricValueList",
    "name":"mock_traffic",
    "namespaced":true,
    "singularName":"",
    "verbs": [
      "get"
    ]
  }
]
}
```

## 配置弹性



## 总结：

基于历史指标预测功能实现原理:
● EHPA开启预测，建立相应指标的时序预测模型【TimeSeriesPrediction】
● 创建HPA，在原有指标基础上，增加时序预测模型指标
● HPA基于metric-adapter服务获取时序预测模型指标，实现服务提前扩容