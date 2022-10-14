---
title: "Introduction"
description: "Main Introduction for Crane"
weight: 10

---

Crane is a FinOps Platform for Cloud Resource Analytics and Economics in Kubernetes clusters. The goal is not only help user to manage cloud cost easier but also ensure the quality of applications.

**How to start a Cost-Saving journey on Crane?**

1. **Understanding**: Cost insight for cloud assets and kubernetes resources(Deployments, StatefulSets).
2. **Analytics**: Periodically analytics the states in cluster and provide optimization recommendations.
3. **Optimization**: Rich set of functionalities to operate and reduce your cost.

<iframe src="https://user-images.githubusercontent.com/35299017/186680122-d7756b47-06be-44cb-8553-1957eaa3ed45.mp4"
scrolling="no" border="0" frameborder="no" framespacing="0" allowfullscreen="true" width="1000" height="600"></iframe>

**Live Demo** for Crane Dashboard: http://dashboard.gocrane.io/

## Main Features

![Crane Overview"](/images/crane-overview.png)

**Cost Visualization and Optimization Evaluation**

- Provides a collection of exporters which collect cloud resource pricing and billing data and ship to your monitoring system like Prometheus.
- Multi-dimensional cost insight, optimization evaluates are supported. Support Multi-cloud Pricing through `Cloud Provider`。

**Recommendation Framework**

Provide a pluggable framework for analytics and give recommendation for cloud resources, support out-of-box recommenders: Workload Resources/Replicas, Idle Resources. [learn more](/docs/tutorials/recommendation).

**Prediction-driven Horizontal Autoscaling**

EffectiveHorizontalPodAutoscaler supports prediction-driven autoscaling. With this capability, user can forecast the incoming peak flow and scale up their application ahead, also user can know when the peak flow will end and scale down their application gracefully. [learn more](/docs/tutorials/using-effective-hpa-to-scaling-with-effectiveness).

**Load-Aware Scheduling**

Provide a simple but efficient scheduler that schedule pods based on actual node utilization data，and filters out those nodes with high load to balance the cluster. [learn more](/docs/tutorials/scheduling-pods-based-on-actual-node-load).

**Colocation with Enhanced QOS**

QOS-related capabilities ensure the running stability of Pods on Kubernetes. It has the ability of interference detection and active avoidance under the condition of multi-dimensional metrics, and supports reasonable operation and custom metrics access; it has the ability to oversell elastic resources enhanced by the prediction algorithm, reuse and limit the idle resources in the cluster; it has the enhanced bypass cpuset Management capabilities, improve resource utilization efficiency while binding cores. [learn more](/docs/tutorials/using-qos-ensurance).

## Architecture

The overall architecture of Crane is shown as below:

![Crane Arch"](/images/crane-arch.png)

**Craned**

Craned is the core component which manage the lifecycle of CRDs and APIs. It's deployed by a `Deployment` which consists of two container:
- Craned: Operators for management CRDs, WebApi for Dashboard, Predictors that provide query TimeSeries API.
- Dashboard: Web component that built from TDesign's Starter, provide an easy-to-use UI for crane users.

**Fadvisor**

Fadvisor provides a collection of exporters which collect cloud resource pricing and billing data and ship to your monitoring system like Prometheus. Fadvisor support Multi-Cloud Pricing API by `Cloud Provider`.

**Metric Adapter**

Metric Adapter implements a `Custom Metric Apiserver`. Metric Adapter consume Crane CRDs and provide HPA Metrics by `Custom/External Metric API`.

**Crane Agent**

Crane Agent is a `DaemonSet` that runs in each node.

## Repositories

Crane is composed of the following components:

- [craned](https://github.com/gocrane/crane/tree/main/cmd/craned) - main crane control plane.
- [metric-adaptor](https://github.com/gocrane/crane/tree/main/cmd/metric-adapter) - Metric server for driving the scaling.
- [crane-agent](https://github.com/gocrane/crane/tree/main/cmd/crane-agent) - Ensure critical workloads SLO based on abnormally detection.
- [gocrane/api](https://github.com/gocrane/api) - This repository defines component-level APIs for the Crane platform.
- [gocrane/fadvisor](https://github.com/gocrane/fadvisor) - Financial advisor which collect resource prices from cloud API.
- [gocrane/crane-scheduler](https://github.com/gocrane/crane-scheduler) - A Kubernetes scheduler which can schedule pod based on actual node load.
