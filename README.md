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
    - [Analytics and Recommendation](#analytics-and-recommendation)
  - [RoadMap](#roadmap)
  - [Contributing](#Contributing)
  - [Code of Conduct](#Code-of-Conduct)

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

<img alt="Crane Overview" height="550" src="docs/images/crane-overview.png" width="800"/>

## Features
### Time Series Prediction

TimeSeriesPrediction defines metric spec to predict kubernetes resources like Pod or Node. 
The prediction module is the core component that other crane components relied on, like [EHPA](#effective-horizontalpodautoscaler) and [Analytics](#analytics).

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
Kubernetes is capable of starting multiple pods on same node, and as a result, some of the user applications may be impacted when there are resources(e.g. cpu) consumption competition. To mitigate this, Crane allows users defining PrioirtyClass for the pods and QoSEnsurancePolicy, and then detects disruption and ensure the high priority pods not being impacted by resource competition.

Avoidance Actions:
- **Disable Schedule**: disable scheduling by setting node taint and condition
- **Throttle**: throttle the low priority pods by squeezing cgroup settings
- **Evict**: evict low priority pods

Please see [this document](./docs/tutorials/using-qos-ensurance.md) to learn more.

## Repositories

Crane is composed of the following components:
- [craned](cmd/craned). - main crane control plane.
  - **Predictor** - Predicts resources metrics trends based on historical data.
  - **AnalyticsController** - Analyzes resources and generate related recommendations.
  - **RecommendationController** - Recommend Pod resource requests and autoscaler.
  - **ClusterNodePredictionController** - Create Predictor for nodes.
  - **EffectiveHPAController** - Effective HPA for horizontal scaling.
  - **EffectiveHPAController** - Effective VPA for vertical scaling.
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
NAME                                             READY   STATUS    RESTARTS   AGE
crane-agent-8h7df                                1/1     Running   0          119m
crane-agent-8qf5n                                1/1     Running   0          119m
crane-agent-h9h5d                                1/1     Running   0          119m
craned-5c69c684d8-dxmhw                          2/2     Running   0          20m
grafana-7fddd867b4-kdxv2                         1/1     Running   0          41m
metric-adapter-94b6f75b-k8h7z                    1/1     Running   0          119m
prometheus-kube-state-metrics-6dbc9cd6c9-dfmkw   1/1     Running   0          45m
prometheus-node-exporter-bfv74                   1/1     Running   0          45m
prometheus-node-exporter-s6zps                   1/1     Running   0          45m
prometheus-node-exporter-x5rnm                   1/1     Running   0          45m
prometheus-server-5966b646fd-g9vxl               2/2     Running   0          45m
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

### Analytics and Recommendation

Crane supports analytics and give recommend advise for your k8s cluster.

Please follow [this guide](./docs/tutorials/analytics-and-recommendation.md) to learn more.

## RoadMap
Please see [this document](./docs/roadmaps/roadmap-1h-2022.md) to learn more.

## Contributing

Contributors are welcomed to join Crane project. Please check [CONTRIBUTING](./CONTRIBUTING.md) about how to contribute to this project.

## Code of Conduct
Crane adopts [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md).