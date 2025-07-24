---
title: "Enhanced bypass cpuset management capability"
description: "Enhanced bypass cpuset management capability"
weight: 23
---

## Enhanced bypass cpuset management capability
Kubelet supports the static CPU manager strategy. When the guaranteed pod runs on the node, kebelet will allocate the specified dedicated CPU for the pod, which cannot be occupied by other processes. This ensures the CPU monopoly of the guaranteed pod, but also causes the low utilization of CPU and nodes, resulting in a certain waste.
Crane agent provides a new strategy for cpuset management, allowing pod and other pod to share CPU. When it specifies CPU binding core, it can make use of the advantages of less context switching and higher cache affinity of binding core, and also allow other workload to deploy and share, so as to improve resource utilization.

1. Three types of pod cpuset are provided:

- Exclusive: after binding the core, other containers can no longer use the CPU and monopolize the CPU
- Share: other containers can use the CPU after binding the core
- None: select the CPU that is not occupied by the container of exclusive pod, can use the binding core of share type

Share type binding strategy can make use of the advantages of less context switching and higher cache affinity, and can also be shared by other workload deployments to improve resource utilization

2. Relax the restrictions on binding cores in kubelet

Originally, it was required that the CPU limit of all containers be equal to the CPU request. Here, it is only required that the CPU limit of any container be greater than or equal to 1 and equal to the CPU request to set the binding core for the container


3. Support modifying the cpuset policy of pod during the running of pod, which will take effect immediately

The CPU manager policy of pod is converted from none to share and from exclusive to share without restart

How to use:
1. Set the cpuset manager of kubelet to "None"
2. Set CPU manager policy through pod annotation
   `qos.gocrane.io/cpu-manager: none/exclusive/share`
   ```yaml
   apiVersion: v1
   kind: Pod
   metadata:
     annotations:
       qos.gocrane.io/cpu-manager: none/exclusive/share
   ```