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
# kubectl get deployment metric-source-service -o yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: metric-source-service
  namespace: default
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
# kubectl get pods -owide
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

### EHPA

#### 设置EHPA

```yaml
apiVersion: autoscaling.crane.io/v1alpha1
kind: EffectiveHorizontalPodAutoscaler
metadata:
  name: metric-source-service
  annotations:
    metric-name.autoscaling.crane.io/mock_traffic: |
#添加注解，当前版本需要配置查询语句
#metric-name.autoscaling.crane.io/${需要获取时序模型的指标名}:
      mock_traffic{job="metrics-service1-lyg-test", pod_project="metric-source-service"}
spec:
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 6
      policies:
      - type: Percent
        value: 100
        periodSeconds: 15
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
      - type: Percent
        value: 100
        periodSeconds: 15
      - type: Pods
        value: 2
        periodSeconds: 15
      selectPolicy: Max
  scaleTargetRef:
#指定需要实现扩缩容的deployment
    apiVersion: apps/v1
    kind: Deployment
    name: metric-source-service
  minReplicas: 2
  maxReplicas: 20
#Auto为应用，Previoew为DryRun模式
  scaleStrategy: Auto
  metrics:
#控制扩缩容的外部指标
  - type: External
    external:
      metric:
        name: mock_traffic
      target:
        averageValue: 2000
        type: AverageValue
#设置时序预测模型
  prediction:
    predictionWindowSeconds: 3600
    predictionAlgorithm:
      algorithmType: dsp
      dsp:
        sampleInterval: "60s"
        historyLength: "7d"
```

应用并查看EHPA状态

```bash
# kubectl apply -f ehpa.yaml
#查询ehpa状态
# kubectl get ehpa
NAME                   STRATEGY   MINPODS   MAXPODS   SPECIFICPODS   REPLICAS   AGE
metric-source-service  Auto       2         20                       10         1m
```

#### 时序预测模型
注：craned定义TimeSeriesPrediction资源作为时序预测模型，命名定义ehpa-${ehpa-name}
增加时序预测指标，命名定义crane-${extrenal-metric-name}

```bash
# kubectl get tsp
NAME                               TARGETREFNAME   TARGETREFKIND   PREDICTIONWINDOWSECONDS   AGE
ehpa-metric-source-service         metric-source-service   Deployment      3600                      1m
# kubectl get tsp ehpa-metric-source-service -o yaml
apiVersion: prediction.crane.io/v1alpha1
kind: TimeSeriesPrediction
metadata:
  name: ehpa-metric-source-service
  namespace: default
spec:
  predictionMetrics:
  - algorithm:
      algorithmType: dsp
      dsp:
        estimators: {}
        historyLength: 7d
        sampleInterval: 60s
    expressionQuery:
      expression: |
        mock_traffic{job="metrics-service1-lyg-test", pod_namespace="cloud", pod_project="me-dingding-service"}
    resourceIdentifier: crane-mock_traffic
    type: ExpressionQuery
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: metric-source-service
    namespace: default
status:
  predictionMetrics:
  - prediction:
    - labels:
      samples:
      - timestamp: 1656402060
        value: "15767.77871"
        ...
        ...
      - timestamp: 1656409200
        value: "22202.37755"
    resourceIdentifier: crane-mock_traffic
```

#### HPA

HPA通过相关指标实现扩缩

```bash
# kubectl get hpa
NAME                  REFERENCE                  TARGETS                                MINPODS   MAXPODS   REPLICAS   AGE
metric-source-service Deployment/metric-source-service   1480200m/2k (avg), 2020300m/2k (avg)   2         20        10         1m
# kubectl describe hpa ehpa-metric-source-service
Name:                                           ehpa-metric-source-service
Namespace:                                      default
Labels:                                         app.kubernetes.io/managed-by=effective-hpa-controller
                                                app.kubernetes.io/name=ehpa-metric-source-service
                                                app.kubernetes.io/part-of=metric-source-service
                                                autoscaling.crane.io/effective-hpa-uid=b2cb76db-61c9-4d00-b333-af67d36bbd65
Annotations:                                    <none>
CreationTimestamp:                              Tue, 28 Jun 2022 13:38:06 +0800
Reference:                                      Deployment/metric-source-service
Metrics:                                        ( current / target )
  "mock_traffic" (target average value):        1470600m / 2k
  "crane-mock_traffic" (target average value):  2032500m / 2k
Min replicas:                                   2
Max replicas:                                   20
Behavior:
  Scale Up:
    Stabilization Window: 0 seconds
    Select Policy: Max
    Policies:
      - Type: Percent  Value: 100  Period: 15 seconds
      - Type: Pods     Value: 2    Period: 15 seconds
  Scale Down:
    Stabilization Window: 6 seconds
    Select Policy: Max
    Policies:
      - Type: Percent  Value: 100  Period: 15 seconds
Deployment pods:       10 current / 10 desired
```

### craned指标采集

#### 副本指标

```bash
# kubectl -n crane-system get deployment craned -o yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: craned
  namespace: crane-system
spec:
  template:
    metadata:
      annotations:
        prometheus.aispeech.com/metric_path: /metrics
        prometheus.aispeech.com/scrape_port: "8080"
        prometheus.aispeech.com/scrape_scheme: http
        prometheus.aispeech.com/should_be_scraped: "true"
 
# kubectl -n crane-system get pods craned-854bcdb88b-d5fgx -o wide
NAME                      READY   STATUS    RESTARTS   AGE   IP             NODE          NOMINATED NODE   READINESS GATES
craned-854bcdb88b-d5fgx   2/2     Running   0          96m   10.244.0.177   d2-node-012   <none>           <none>
#指标查询
# curl -sL 10.244.0.177:8080/metrics | grep ehpa
crane_autoscaling_effective_hpa_replicas{name="metric-source-service",namespace="default"} 10
crane_autoscaling_hpa_replicas{name="ehpa-metric-source-service",namespace="default"} 10
crane_autoscaling_hpa_scale_count{name="ehpa-metric-source-service",namespace="default",type="hpa"} 3
```

#### TSP指标

```bash
# curl -sL 10.244.0.177:8080/metrics | grep ^crane_prediction_time_series_prediction
crane_prediction_time_series_prediction_external_by_window{algorithm="dsp",resourceIdentifier="crane-mock_traffic",targetKind="Deployment",targetName="metric-source-service",targetNamespace="default",type="ExpressionQuery"} 23011 1657270905000
crane_prediction_time_series_prediction_resource{algorithm="dsp",resourceIdentifier="crane_pod_cpu_usage",targetKind="Deployment",targetName="metric-source-service",namespace="default"} 10
```

## 总结：

基于历史指标预测功能实现原理:
● EHPA开启预测，建立相应指标的时序预测模型【TimeSeriesPrediction】
● 创建HPA，在原有指标基础上，增加时序预测模型指标
● HPA基于metric-adapter服务获取时序预测模型指标，实现服务提前扩容
