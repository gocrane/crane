# Crane: Cloud Resource Analytics and Economics

[![Go Report Card](https://goreportcard.com/badge/github.com/gocrane/crane)](https://goreportcard.com/report/github.com/gocrane/crane)
[![GoDoc](https://godoc.org/github.com/gocrane/crane?status.svg)](https://godoc.org/github.com/gocrane/crane)
[![License](https://img.shields.io/github/license/gocrane/crane)](https://www.apache.org/licenses/LICENSE-2.0.html)
![GoVersion](https://img.shields.io/github/go-mod/go-version/gocrane/crane)

<img alt="Crane logo" height="100" src="docs/images/crane.svg" title="Crane" width="200"/>

---

Crane (FinOps Crane) is a cloud native open source project which manages cloud resources on Kubernetes stack, it is inspired by FinOps concepts.

- [Crane: Cloud Resource Analytics and Economics](#crane-cloud-resource-analytics-and-economics)
  - [Introduction](#introduction)
  - [Features](#features)
    - [TimeSeriesPrediction](#Time-series-prediction)
    - [Effective HorizontalPodAutoscaler](#effective-horizontalpodautoscaler)
    - [Analytics](#analytics)
    - [QoS Ensurance](#qos-ensurance)
  - [Repositories](#repositories)
  - [Getting Started](#getting-started)
    - [Installation](#installation)
    - [Get your Kubernetes Cost Report](#get-your-kubernetes-cost-report)
    - [Analytics and Recommend Pod Resources](#analytics-and-recommend-pod-resources)
    - [Analytics and Recommend HPA](#analytics-and-recommend-hpa)
  - [RoadMap](#roadmap)

## Introduction

The goal of Crane is to provide a one-stop-shop project to help Kubernetes users to save cloud resource usage with a rich set of functionalities:

- **Time Series Prediction** based on monitoring data
- **Usage and Cost visibility**
- **Usage & Cost Optimization** including:
  - R2 (Resource Re-allocation)
  - R3 (Request & Replicas Recommendation)
  - Effective Pod Autoscaling (Effective Horizontal & Vertical Pod Autoscaling)
  - Cost Optimization
- **Enhanced QoS** based on Pod PriorityClass

<img alt="Crane Overview" height="700" src="docs/images/crane-overview.png" width="886"/>

## Features
### Time Series Prediction

Crane predictor fetches metric data, and then outputs the prediction results.
The prediction result can be consumed by other crane components, like [EHPA](#effective-horizontalpodautoscaler) and [Analytics](#analytics).

Please see [this document](./docs/tutorials/using-time-series-prediction.md) to learn more.

### Effective HorizontalPodAutoscaler

EffectiveHorizontalPodAutoscaler helps you manage application scaling in an easy way. It is compatible with native [HorizontalPodAutoscaler](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/) but extends more features like prediction-driven autoscaling.

Please see [this document](./docs/tutorials/using-effective-hpa-to-scaling-with-effectiveness.md) to learn more.

### Analytics

Analytics model analyzes the workload and provide recommendations about resource optimize.

Two Recommendations are currently supported:
- **ResourceRecommend**: Recommend container requests & limit resources based on historic metrics.
- **Effective HPARecommend**: Recommend which workloads are suitable for autoscaling and provide optimized configurations such as minReplicas, maxReplicas.

### QoS Ensurance

## Repositories

Crane is composed of the following components:
- [craned](cmd/craned). - main crane control plane.
  - **Predictor** - Predicts resources metrics trends based on historical data.
  - **AnalyticsController** - Analyzes resources and generate related recommendations.
  - **RecommendationController** - Recommend Pod resource requests and autoscaler.
  - **NodeResourceController** - Re-allocate node resource based on prediction result.
  - **EffectiveHPAController** - Effective HPA based on prediction result.
- [metric-adaptor](cmd/metric-adapter). - Metric server for driving the scaling.
- [crane-agent](cmd/crane-agent). - Ensure critical workloads SLO based on abnormally detection.
- [gocrane/api](https://github.com/gocrane/api). This repository defines component-level APIs for the Crane platform.
- [gocrane/fadvisor](https://github.com/gocrane/fadvisor) Financial advisor which collect resource prices from cloud API. 

## Getting Started

### Installation

**Prerequisites**

- Kubernetes 1.18+
- Helm 3.1.0

**Helm Installation**

Please refer to Helm's [documentation](https://helm.sh/docs/intro/install/) for installation.

**Installing prometheus and grafana with helm chart**

> Note:
> If you already deployed prometheus, grafana in your environment, then skip this step.

Crane use prometheus to be the default metric provider. Using following command to install prometheus components: prometheus-server, node-exporter, kube-state-metrics.

```console
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus -n crane-system --set pushgateway.enabled=false --set alertmanager.enabled=false --set server.persistentVolume.enabled=false -f https://raw.githubusercontent.com/gocrane/helm-charts/main/integration/prometheus/override_values.yaml --create-namespace  prometheus-community/prometheus
```

Fadvisor use grafana to present cost estimates. Using following command to install a grafana.

```console
helm repo add grafana https://grafana.github.io/helm-charts
helm install grafana -f https://raw.githubusercontent.com/gocrane/helm-charts/main/integration/grafana/override_values.yaml -n crane-system --create-namespace grafana/grafana
```

**Deploying Crane and Fadvisor**

```console
helm repo add crane https://gocrane.github.io/helm-charts
helm install crane -n crane-system --create-namespace crane/crane
helm install fadvisor -n crane-system --create-namespace crane/fadvisor
```

**Verify Installation**

Check deployments are all available by running:

```console
kubectl get deploy -n crane-system
```

The output is similar to:
```console
NAME                            READY   UP-TO-DATE   AVAILABLE   AGE
craned                          1/1     1            1           60m
fadvisor                        1/1     1            1           60m
grafana                         1/1     1            1           60m
metric-adapter                  1/1     1            1           60m
prometheus-kube-state-metrics   1/1     1            1           61m
prometheus-server               1/1     1            1           61m
```

you can see [this](https://github.com/gocrane/helm-charts) to learn more.

**Customize Installation**

Deploy `Crane` by apply YAML declaration.

```console
git checkout v0.2.0
kubectl apply -f deploy/manifests 
kubectl apply -f deploy/craned 
kubectl apply -f deploy/metric-adapter
```

The following command will configure prometheus http address for crane if you want to customize it. Specify `CUSTOMIZE_PROMETHEUS` if you have existing prometheus server.

```console
export CUSTOMIZE_PROMETHEUS=
if [ $CUSTOMIZE_PROMETHEUS ]; then sed -i '' "s/http:\/\/prometheus-server.crane-system.svc.cluster.local:8080/${CUSTOMIZE_PROMETHEUS}/" deploy/craned/deployment.yaml ; fi
```

### Get your Kubernetes Cost Report

Get the Grafana URL to visit by running these commands in the same shell:

```console
export POD_NAME=$(kubectl get pods --namespace crane-system -l "app.kubernetes.io/name=grafana,app.kubernetes.io/instance=grafana" -o jsonpath="{.items[0].metadata.name}")
kubectl --namespace crane-system port-forward $POD_NAME 3000
```

visit [Cost Report](http://127.0.0.1:3000/dashboards) here with account(admin:admin).

### Analytics and Recommend Pod Resources

Create an **Resource** `Analytics` to give recommendation for deployment: `craned` and `metric-adapter` as a sample.

```console
kubectl apply -f https://raw.githubusercontent.com/gocrane/crane/main/examples/analytics/analytics-resource.yaml
kubectl get analytics -n crane-system
```

The output is:

```console
NAME                      AGE
craned-resource           15m
metric-adapter-resource   15m
```

You can get created recommendation from analytics status:

```console
kubectl get analytics craned-resource -n crane-system -o yaml
```

The output is similar to:

```console 
apiVersion: analysis.crane.io/v1alpha1
kind: Analytics
metadata:
  name: craned-resource
  namespace: crane-system
spec:
  completionStrategy:
    completionStrategyType: Periodical
    periodSeconds: 86400
  resourceSelectors:
  - apiVersion: apps/v1
    kind: Deployment
    labelSelector: {}
    name: craned
  type: Resource
status:
  lastSuccessfulTime: "2022-01-12T08:40:59Z"
  recommendations:
  - name: craned-resource-resource-j7shb
    namespace: crane-system
    uid: 8ce2eedc-7969-4b80-8aee-fd4a98d6a8b6    
```

The recommendation name presents on `status.recommendations[0].name`. Then you can get recommendation detail by running:

```console
kubectl get recommend -n crane-system craned-resource-resource-j7shb -o yaml
```

The output is similar to:

```console
apiVersion: analysis.crane.io/v1alpha1
kind: Recommendation
metadata:
  name: craned-resource-resource-j7shb
  namespace: crane-system
  ownerReferences:
  - apiVersion: analysis.crane.io/v1alpha1
    blockOwnerDeletion: false
    controller: false
    kind: Analytics
    name: craned-resource
    uid: a9e6dc0d-ab26-4f2a-84bd-4fe9e0f3e105
spec:
  completionStrategy:
    completionStrategyType: Periodical
    periodSeconds: 86400
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: craned
    namespace: crane-system
  type: Resource
status:
  conditions:
  - lastTransitionTime: "2022-01-12T08:40:59Z"
    message: Recommendation is ready
    reason: RecommendationReady
    status: "True"
    type: Ready
  lastSuccessfulTime: "2022-01-12T08:40:59Z"
  lastUpdateTime: "2022-01-12T08:40:59Z"
  resourceRequest:
    containers:
    - containerName: craned
      target:
        cpu: 114m
        memory: 120586239m
```

The `status.resourceRequest` is recommended by crane's recommendation engine.
 
Something you should know about Resource recommendation:
* Resource Recommendation use historic prometheus metrics to calculate and propose.
* We use **Percentile** algorithm to process metrics that also used by VPA.
* If the workload is running for a long term like several weeks, the result will be more accurate.

### Analytics and Recommend HPA

Create an **HPA** `Analytics` to give recommendation for deployment: `craned` and `metric-adapter` as an sample.

```console
kubectl apply -f https://raw.githubusercontent.com/gocrane/crane/main/examples/analytics/analytics-hpa.yaml
kubectl get analytics -n crane-system 
```

The output is:

```console
NAME                      AGE
craned-hpa                5m52s
craned-resource           18h
metric-adapter-hpa        5m52s
metric-adapter-resource   18h

```

You can get created recommendation from analytics status:

```console
kubectl get analytics craned-hpa -n crane-system -o yaml
```

The output is similar to:

```console 
apiVersion: analysis.crane.io/v1alpha1
kind: Analytics
metadata:
  name: craned-hpa
  namespace: crane-system
spec:
  completionStrategy:
    completionStrategyType: Periodical
    periodSeconds: 86400
  resourceSelectors:
  - apiVersion: apps/v1
    kind: Deployment
    labelSelector: {}
    name: craned
  type: HPA
status:
  lastSuccessfulTime: "2022-01-13T07:26:18Z"
  recommendations:
  - apiVersion: analysis.crane.io/v1alpha1
    kind: Recommendation
    name: craned-hpa-hpa-2f22w
    namespace: crane-system
    uid: 397733ee-986a-4630-af75-736d2b58bfac
```

The recommendation name presents on `status.recommendations[0].name`. Then you can get recommendation detail by running:

```console
kubectl get recommend -n crane-system craned-resource-resource-j7shb -o yaml
```

The output is similar to:

```console
apiVersion: analysis.crane.io/v1alpha1
kind: Recommendation
metadata:
  name: craned-hpa-hpa-2f22w
  namespace: crane-system
  ownerReferences:
  - apiVersion: analysis.crane.io/v1alpha1
    blockOwnerDeletion: false
    controller: false
    kind: Analytics
    name: craned-hpa
    uid: b216d9c3-c52e-4c9c-b9e9-9d5b45165b1d
spec:
  completionStrategy:
    completionStrategyType: Periodical
    periodSeconds: 86400
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: craned
    namespace: crane-system
  type: HPA
status:
  conditions:
  - lastTransitionTime: "2022-01-13T07:51:18Z"
    message: 'Failed to offer recommend, Recommendation crane-system/craned-hpa-hpa-2f22w
      error EHPAAdvisor prediction metrics data is unexpected, List length is 0 '
    reason: FailedOfferRecommend
    status: "False"
    type: Ready
  lastUpdateTime: "2022-01-13T07:51:18Z"
```

The `status.resourceRequest` is recommended by crane's recommendation engine. The fail reason is demo workload don't have enough run time.

Something you should know about HPA recommendation:
* HPA Recommendation use historic prometheus metrics to calculate, forecast and propose.
* We use **DSP** algorithm to process metrics.
* We recommend using Effective HorizontalPodAutoscaler to execute autoscaling, you can see [this document](./docs/tutorials/using-time-series-prediction.md) to learn more.
* The Workload need match following conditions:
  * Existing at least one ready pod
  * Ready pod ratio should larger that 50%
  * Must provide cpu request for pod spec
  * The workload should be running for at least **a week** to get enough metrics to forecast
  * The workload's cpu load should be predictable, **too low** or **too unstable** workload often is unpredictable

## RoadMap
Please see [this document](./docs/roadmaps/roadmap-1h-2022.md) to learn more.

