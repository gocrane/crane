---
title: "Architecture"
description: "Overall Architecture"
weight: 18

---

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
- [kubectl-crane](https://github.com/gocrane/kubectl-crane) - Kubectl plugin for crane, including recommendation and cost estimate.
