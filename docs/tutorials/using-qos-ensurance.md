# Qos Ensurance
QoS ensurance guarantees the stability of the pods running on Kubernetes.

It has the ability of interference detection and active avoidance. When pod with higher priority is affected by resource competition, disable schedule, throttle and evict will be applied to pod with lower priority to ensure the overall stability of the node. Currently, the absolute value/percentage of the node's cpu/mem is supported as a watermark. For details, please refer to qos-interference-detection-and-active-avoidance.md.
When there is interference for eviction or throttle, accurate calculation will be performed, and the operation will be stopped when the load is lowered to slightly below the watermark to prevent accidental injury and transitional operation. For details, please refer to qos-accurately-perform-avoidance-actions.md.

At the same time, Crane supports custom metrics to adapt to the entire interference detection framework. Users only need to complete some operations such as sorting definition, and can reuse the interference detection and avoidance processes including precise operations.For details, please refer to qos-customized-metrics-interference-detection-avoidance-and-sorting.md.

Crane has the dynamic resource oversold ability enhanced by the prediction algorithm, and reuses the idle resources. At the same time, it combines the prediction ability of the crane to better reuse the idle resources. Currently, idle resource recycling of cpu and mem is supported. At the same time, it has the elastic resource limitation function to limit the workload of reusing idle resources, and avoid impact on high-quality business and starvation issues. For details, please refer to qos-dynamic-resource-oversold-and-limit.md.

At the same time, it has enhanced bypass cpuset management capability to improve resource utilization efficiency while binding cores. For details, please refer to qos-enhanced-bypass-cpuset-management.md.