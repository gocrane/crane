# Recommendation Framework Internal

## Summary

This document describes the Crane Recommendation Framework Internal. We will propose the four major modules of Crane Recommendation in this proposal. By clearly dividing the functions of the modules and defining the interface, developers can expand the recommendation more conveniently and flexibly.

## Motivation

At present, crane Recommendation has been applied to kubernetes resource fields such as resource recommendation, replica recommendation, HPA recommendation, etc. The algorithm modules of crane, such as DSP, Max and Percentile algorithm modules, have been verified to be stable and effective in production practice.At the same time, the offline data source of crane supports prometheus, grpc protocol service, and the online data source supports prometheus and metricsserver. However, we have received a lot of feedback from developers, mainly focusing on the following aspects:

1. After I have defined a lot of Recommendations, I don't know which recommendation results are accurate. I want to add some filtering logic, but there seems to be no such interface.
2. Our monitoring system is not in the default implementation, how can I implement a custom interface so that my resources can also use crane's recommended optimization capabilities?
3. We found that the crane algorithm is not very effective for our business type, but we have explored some effective algorithms before, how to connect to the crane system?
4. We want to be able to interface directly to the billing system after cost optimization, so we can directly quantify how much money is saved.

In order to solve the above problems, we hope the whole recommendation process is more open and flexible. Therefore, we propose the crane recommendation framework, which will be divided into two types. The first is to implement recommendation flow logic in crane code, and the second is out-of-tree, you need to implement extension point through http request. This documentation will focus on the first implementation type.

## Goals

- Define the architecture of Recommendation Framework.
- Define the interfaces of Recomendation Framework Internal modules.

## Non-Goals

- Define the interfaces of Recommendation Framework Extender.
- Provide specific implementation examples for each module of framework.

## Proposal

### Architecture

![](../images/crane_recommendation_framework.jpg)