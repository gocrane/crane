# Recommendation Definition
- This proposal explains the metrics challenges we encountered in production and proposal an enhanced solution.  

## Table of Contents

## Motivation
In a production cluster, we got alerts that some node cpu utilization are above the threshold but new pods are still bound to it. After the investigation we found the root cause:
- Crane-scheduler is using monitoring system as the source of truth of the node load, as the monitoring system like prometheus has a 30s pull interval, so there is a latency between actual overload and monitoring system.
- The annotator updates the latest load to node annotations by a fixed interval which is default to 5m

With all above latencies, when user deploys a workload into the cluster, it is very possible that multiple pods are scheduled to same node and make the node being overload. And because of the metrics and annotator latency, new pending pods would still be scheduled to it.  

## Proposal
This proposal describe a solution which reduce the latency as much as possible, so crane-scheduler can catch the up-to-date load changes on the nodes and make correct scheduling decision.
- Current crane scheduler support 3 interval metrics, 1d, 1h and 5m, it makes sense to keep 1d and 1h metrics to be retrieved from monitoring system, however latency of 5m metrics caused problem mentioned above. So it is proposed that crane-scheduler should retrieve the short interval metrics from metrics server.  
- As Crane Recommendation can provide resource recommendation on workload, which is Pxx load of the pending pods, so we should enhance the dynamic scheduling plugin, a node can be scheduling candidate only if the following statement is true(taking cpu as the example): 
```
node cpu second total + sum(pendingpod.recommendation.status.resourceRequest.target.cpu) <= node allocatable cpu * target utilization  
```
- If there is no recommendation for a pod, for example the pod does not belong to any existing workload, and there is no existing metrics for craned to predict the usage and give recommendation, the scheduler should just use pod request as the recommended value during scheduling.
```
node cpu second total + sum(pendingpod.resources.requests.cpu) <= node allocatable cpu * target utilization
```

## Benefits
Remove the latency between scheduler and the actual usage change, avoid node being overloaded.