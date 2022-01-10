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

<img alt="Crane Overview" height="400" src="docs/images/crane-overview.png" width="700"/>

## Features
### Time Series Prediction

Crane predictor fetches metric data, and then outputs the prediction results.
The prediction result can be consumed by other crane components, like [EHPA](#effective-horizontalpodautoscaler) and [Analytics](#analytics).

Please see [this document](./docs/tutorials/using-time-series-prediction.md) to learn more.

### Effective HorizontalPodAutoscaler

EffectiveHorizontalPodAutoscaler helps you manage application scaling in an easy way. It is compatible with native [HorizontalPodAutoscaler](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/) but extends more features.

Please see [this document](./docs/tutorials/using-effective-hpa-to-scaling-with-effectiveness.md) to learn more.

### Analytics

Analytics model analyzes the workload and provide recommendations about resource optimize.

Two Recommendations are currently supported:
- **ResourceRecommend**: Recommend container requests & limit resources based on historic metrics.
- **Effective HPARecommend**: Recommend which workloads are suitable for autoscaling and provide optimized configurations such as minReplicas, maxReplicas.

### QoS Ensurance
Qos ensurance was used to protect the  high level service's quality not be effected. It detects the load of the node and the service metrics in real time.
When found the high level service be effected, we will throttle or evict low level service's pods to ensurance the high level service as far as possible.

Supported detected source:
- **NodeLocal**: node resource metrics
- **ApplicationMetrics**: application metrics (planning)

Supported actions:
- **Disable Schedule**: disable schedule by set taint and condition
- **Throttle**: throttle the low level service's pods
- **Evict**: evict the low level service' pods

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