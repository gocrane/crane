# Qos Ensurance
Qos Ensurance 保证了运行在 Kubernetes 上的 Pod 的稳定性。

具有干扰检测和主动回避能力，当较高优先级的 Pod 受到资源竞争的影响时，Disable Schedule、Throttle以及Evict 将应用于低优先级的 Pod，以保证节点整体的稳定，
目前已经支持节点的cpu/mem负载绝对值/百分比作为水位线，具体可以参考[干扰检测和主动回避](qos-interference-detection-and-active-avoidance.zh.md)
在发生干扰进行驱逐或压制时，会进行精确计算，将负载降低到略低于水位线即停止操作，防止误伤和过渡操作，具体内容可以参照[精确执行回避动作](qos-accurately-perform-avoidance-actions.zh.md)。

同时，crane支持自定义指标适配整个干扰检测框架，只需要完成排序定义等一些操作，即可复用包含精确操作在内的干扰检测和回避流程，具体内容可以参照[定义自己的水位线指标](qos-customized-metrics-interference-detection-avoidance-and-sorting.zh.md)。

具有预测算法增强的弹性资源超卖能力，将集群内的空闲资源复用起来，同时结合crane的预测能力，更好地复用闲置资源，当前已经支持cpu和mem的空闲资源回收。同时具有弹性资源限制功能，限制使用弹性资源的workload最大和最小资源使用量，避免对高优业务的影响和饥饿问题。具体内容可以参照[弹性资源超卖和限制](qos-dynamic-resource-oversold-and-limit.zh.md)。

同时具备增强的旁路cpuset管理能力，在绑核的同时提升资源利用效率，具体内容可以参照[增强的旁路cpuset管理能力](qos-enhanced-bypass-cpuset-management.zh.md)。