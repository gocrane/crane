---
title: "Introduction"
description: "Main Introduction for Crane"
weight: 10

---

Crane is a FinOps Platform for Cloud Resource Analytics and Economics in Kubernetes clusters. The goal is not only help user to manage cloud cost easier but also ensure the quality of applications.

<img alt="fcs logo" height="200" src="/images/Crane-FinOps-Certified-Solution.png" title="FinOps Certified Solution" width="200"/>

Crane is a [FinOps Certified Solution](https://www.finops.org/members/finops-certified-solution/) project of the [FinOps Foundation](https://www.finops.org/).

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

Provide a pluggable framework for analytics and give recommendation for cloud resources, support out-of-box recommenders: Workload Resources/Replicas/HPA, Idle Resources. [learn more](/docs/tutorials/recommendation).

**Prediction-driven Horizontal Autoscaling**

EffectiveHorizontalPodAutoscaler supports prediction-driven autoscaling. With this capability, user can forecast the incoming peak flow and scale up their application ahead, also user can know when the peak flow will end and scale down their application gracefully. [learn more](/docs/tutorials/using-effective-hpa-to-scaling-with-effectiveness).

**Load-Aware Scheduling**

Provide a simple but efficient scheduler that schedule pods based on actual node utilization data，and filters out those nodes with high load to balance the cluster. [learn more](/docs/tutorials/scheduling-pods-based-on-actual-node-load).

**Colocation with Enhanced QOS**

QOS-related capabilities ensure the running stability of Pods on Kubernetes. It has the ability of interference detection and active avoidance under the condition of multi-dimensional metrics, and supports reasonable operation and custom metrics access; it has the ability to oversell elastic resources enhanced by the prediction algorithm, reuse and limit the idle resources in the cluster; it has the enhanced bypass cpuset Management capabilities, improve resource utilization efficiency while binding cores. [learn more](/docs/tutorials/using-qos-ensurance).
