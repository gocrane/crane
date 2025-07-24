---
title: "Application Resource Optimize Model"
description: "Application Resource Optimize Model"
weight: 11

---

Resource optimization is a common optimization strategy in FinOps. Based on the characteristics of Kubernetes applications, we have summarized the **resource optimization model** for cloud-native applications:

![Resource Model](/images/resource-model.png)

The five lines in the figure from top to bottom are:

1. Node capacity: the total resources of all nodes in the cluster, corresponding to the Capacity of the cluster
2. Allocated: the total resources applied by the application, corresponding to Pod Request
3. Weekly peak usage: the peak resource usage of the application in the past period. The weekly peak can predict the resource usage in the future period. Configuring resource specifications based on weekly peak can ensure higher security and stronger versatility.
4. Daily peak usage: the peak resource usage of the application in one day
5. Average usage: the average resource usage of the application, corresponding to Usage

There are two types of idle resources:

1. **Resource Slack**: the difference between Capacity and Request
2. **Usage Slack**: the difference between Request and Usage

Total Slack = Resource Slack + Usage Slack

The goal of resource optimization is to reduce Resource Slack and Usage Slack. The model provides four steps for reducing waste, from top to bottom:

1. Improve packing rate: improving packing rate can make Capacity and Request closer. There are many methods, such as [dynamic scheduler](/docs/tutorials/scheduling-pods-based-on-actual-node-load), Tencent Cloud native node's node enlargement function, etc.
2. Adjust application requests to reduce resource locking: adjust application specifications based on the weekly peak resource usage to reduce Request to the weekly peak line. [Resource recommendation](/docs/tutorials/recommendation/resource-recommendation) and [replica recommendation](/docs/tutorials/recommendation/replicas-recommendation) can help applications achieve this goal.
3. Application requests adjustment + scaling to cope with sudden traffic bursts: based on the request optimization, use HPA to cope with sudden traffic bursts, and reduce Request to the daily average peak line. At this time, the target utilization rate of HPA is low, only for coping with sudden traffic, and autoscaling does not occur most of the time. [HPA recommendation](/docs/tutorials/recommendation/hpa-recommendation) can scan out applications suitable for elasticity and provide HPA configuration.
4. Application requests adjustment + scaling to cope with daily traffic changes: based on the request optimization, use HPA to cope with daily traffic and reduce Request to the average. At this time, the target utilization rate of HPA is equal to the average utilization rate of the application. [EHPA](/docs/tutorials/using-effective-hpa-to-scaling-with-effectiveness) provide prediction-based horizontal elasticity, helping more applications achieve intelligent elasticity.
