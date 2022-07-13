# Recommendation Framework Extender

## Summary

This document describes the Crane Recommendation Framework Extender.At present, crane has not provided a mechanism for non-invasive function realization. If the user needs to implement an internal recommendation function, the source code of crane must be modified.Such as kube-scheduler, this proposal is to dynamically extend crane's recommendation capabilities via an http request, while keeping the recommendation "core" simple and maintainable.

## Motivation

In crane, we have defined `Analytics` and `Recommendation` to provide a recommendation service for workload's resources. We aim to reduce the cost by analyzing the specific indicators of the kubernetes workload to recommend better configurations. As of the crane v0.6.0, we support replicas and resource(cpu,memory) recommendation. In the future, we hope to analyze more monitoring data and expand the recommendation task, such as load balance, vm type or gpu type. But when we want to further expand our recommendation task with our community partners, we will encounter the following problems:
- The recommended analysis tasks are based on historical time series, and currently users use different tdsb, such as open source software prometheus, Influxdb or public cloud monitoring services, such as AWS's cloudwatch or Tencent Cloud's cloud monitoring. So many different and similar data access layers are placed in the core code of Crane, maintenance and usability will encounter huge challenges.
- Crane's recommendation algorithm is constantly updated. For different tasks, the optimal algorithm may also be different, especially the strategies for different tasks are also different.For example, the percentile algorithm for request recommendation needs to set margin to ensure the quality of service, or set the strategy of the target utilization for replicas recommendation.In order to ensure the flexibility and evaluability of algorithms and strategies, we want the algorithm and strategy layers to be pluggable.
- Many users have reported that after getting the recommended results of crane, they need to write some programs to organize the data to get the results they want. For example, for the calculation of the number of copies and the cost savings after resource recommendation, we hope to provide Plug-in extensible interface that allows users to customize the program to consume the recommended results.

Currently, the only way to implement a recommendation flow based on crane is to modify crane's code and recompile.This method requires developers to deeply understand the working principle of crane and the related architecture, which is not friendly to beginners. By adding other processes and based on network communication, it will bring some performance degradation, but it will also improve scalability. Users can make their own trade-offs between the two.


## Goals

- Make recommendation flow more extendable.
- Propose extension points in the framework.
- Propose a mechanism to receive plugin results and continue or abort based on the received results.
- Propose a mechanism to handle errors and communicate them with plugins.

## Non-Goals

- Solve all recommendation task.
- Provide implementation details of plugins and call-back functions, such as all of their arguments and return values.
- Provide non-kubernetes resource recommendation support.

## Proposal

The Recommendation Framework defines new extension points and Go APIs in the Crane Recommendation for use by "plugins". Plugins add recommendation behaviors to the crane, and are included at compile time. The recommendation's ComponentConfig will allow plugins to be enabled, disabled, and reordered. Custom recommendations can write their plugins "out-of-tree" and compile a craned binary with their own plugins included.

### Extension points

The following picture shows the recommendation context of a recommendation task and the extension points that the recommendation framework exposes. Plugins are registered to be called at one or more of these extension points. In the following sections we describe each extension point in the same order they are called.

#### PreFilter

PreFliter plugin is used to sort recommendation object in the recommendation queue. A queue sort plugin essentially will provide a "less(recommendation1, recommendation2)" function. Only one queue sort PreFliter plugin may be enabled at a time.


#### Fliter

A filter plugin should implement a Filter function, if Filter returns an error, the recommendation cycle is aborted. Note that Filter is called once in each scheduling cycle.A filter plugin can implement the optional `FilterExtensions` interface which define AddRecommendation and RemoveRecommendation methods to incrementally modify its pre-processed info. 

#### PrePrepare

