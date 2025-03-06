package metrics

import (
	"github.com/gocrane/crane/pkg/known"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	v1 "k8s.io/client-go/listers/core/v1"
	k8smetrics "k8s.io/component-base/metrics"
	"k8s.io/klog/v2"
)

const (
	CraneNodeSubsystem = "node"
	CranePodSubsystem  = "pod"
)

var (
	podElasticCPUDesc = k8smetrics.NewDesc("crane_pod_elastic_cpu_request",
		"The elastic cpu requested by pod",
		[]string{"pod", "namespace"},
		nil,
		k8smetrics.ALPHA,
		"",
	)
	podElasticMemoryDesc = k8smetrics.NewDesc("crane_pod_elastic_memory_request",
		"The elastic cpu requested by pod",
		[]string{"pod", "namespace"},
		nil,
		k8smetrics.ALPHA,
		"",
	)
)

func NewPodResourceCollector(podLister v1.PodLister) *PodResourceCollector {
	return &PodResourceCollector{
		podLister: podLister,
	}
}

type PodResourceCollector struct {
	k8smetrics.BaseStableCollector
	podLister v1.PodLister
}

func (n *PodResourceCollector) DescribeWithStability(descs chan<- *k8smetrics.Desc) {
	descs <- podElasticCPUDesc
	descs <- podElasticMemoryDesc
}

func (n *PodResourceCollector) CollectWithStability(metrics chan<- k8smetrics.Metric) {
	pods, err := n.podLister.List(labels.Everything())
	if err != nil {
		klog.ErrorS(err, "list pods failed")
		return
	}

	for _, pod := range pods {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}
		eCPU, eMemory := resource.NewQuantity(0, resource.DecimalSI), resource.NewQuantity(0, resource.BinarySI)
		for _, container := range pod.Spec.Containers {
			eCPU.Add(*container.Resources.Requests.Name(known.ElasticCPU, resource.DecimalSI))
			eMemory.Add(*container.Resources.Requests.Name(known.ElasticMemory, resource.DecimalSI))
		}
		if eCPU.IsZero() && eMemory.IsZero() {
			continue
		}
		metrics <- k8smetrics.NewLazyConstMetric(podElasticCPUDesc, k8smetrics.GaugeValue, eCPU.AsApproximateFloat64(), pod.Name, pod.Namespace)
		metrics <- k8smetrics.NewLazyConstMetric(podElasticMemoryDesc, k8smetrics.GaugeValue, eMemory.AsApproximateFloat64(), pod.Name, pod.Namespace)
	}
}

var (
	nodeElasticCPUDesc = k8smetrics.NewDesc("crane_node_elastic_cpu_allocatable",
		"The elastic cpu of the node.",
		[]string{"node"},
		nil,
		k8smetrics.ALPHA,
		"",
	)
	nodeElasticMemoryDesc = k8smetrics.NewDesc("crane_node_elastic_memory_allocatable",
		"The elastic memory requested by pod",
		[]string{"node"},
		nil,
		k8smetrics.ALPHA,
		"",
	)
	nodeCPUAllocatableDesc = k8smetrics.NewDesc("crane_node_cpu_allocatable",
		"The cpu allocatable of the node.",
		[]string{"node"},
		nil,
		k8smetrics.ALPHA,
		"",
	)
	nodeCPUCapacityDesc = k8smetrics.NewDesc("crane_node_cpu_capacity",
		"The cpu capacity of the node.",
		[]string{"node"},
		nil,
		k8smetrics.ALPHA,
		"",
	)
	nodeMemoryAllocatableDesc = k8smetrics.NewDesc("crane_node_memory_allocatable",
		"The memory allocatable requested by pod",
		[]string{"node"},
		nil,
		k8smetrics.ALPHA,
		"",
	)
	nodeMemoryCapacityDesc = k8smetrics.NewDesc("crane_node_memory_capacity",
		"The memory capacity requested by pod",
		[]string{"node"},
		nil,
		k8smetrics.ALPHA,
		"",
	)
	nodeCPUReservedDesc = k8smetrics.NewDesc("crane_node_cpu_reserved",
		"The reserved cpu of node",
		[]string{"node"},
		nil,
		k8smetrics.ALPHA,
		"",
	)
	nodeCPUUsageOnlineDesc = k8smetrics.NewDesc("crane_node_cpu_usage_online",
		"The online cpu usage of node",
		[]string{"node"},
		nil,
		k8smetrics.ALPHA,
		"",
	)
	nodeCPUUsageOfflineDesc = k8smetrics.NewDesc("crane_node_cpu_usage_offline",
		"The offline cpu usage of node",
		[]string{"node"},
		nil,
		k8smetrics.ALPHA,
		"",
	)
	nodeMemoryReservedDesc = k8smetrics.NewDesc("crane_node_memory_reserved",
		"The reserved memory of node",
		[]string{"node"},
		nil,
		k8smetrics.ALPHA,
		"",
	)
	nodeMemoryUsageOnlineDesc = k8smetrics.NewDesc("crane_node_memory_usage_online",
		"The online memory usage of node",
		[]string{"node"},
		nil,
		k8smetrics.ALPHA,
		"",
	)
	nodeMemoryUsageOfflineDesc = k8smetrics.NewDesc("crane_node_memory_usage_offline",
		"The offline memory usage of node",
		[]string{"node"},
		nil,
		k8smetrics.ALPHA,
		"",
	)
)

type NodeResourceCollector struct {
	k8smetrics.BaseStableCollector
	nodeName           string
	nodeLister         v1.NodeLister
	nodeResourceGetter func() *known.ResourceStatus
}

func NewNodeResourceCollector(nodeName string, nodeLister v1.NodeLister, nodeResourceGetter func() *known.ResourceStatus) *NodeResourceCollector {
	return &NodeResourceCollector{
		nodeName:           nodeName,
		nodeLister:         nodeLister,
		nodeResourceGetter: nodeResourceGetter,
	}
}

func (n *NodeResourceCollector) DescribeWithStability(descs chan<- *k8smetrics.Desc) {
	// resource metrics from status of node
	descs <- nodeElasticCPUDesc
	descs <- nodeElasticMemoryDesc
	descs <- nodeCPUAllocatableDesc
	descs <- nodeCPUCapacityDesc
	descs <- nodeMemoryAllocatableDesc
	descs <- nodeMemoryCapacityDesc

	// usage metrics
	descs <- nodeCPUReservedDesc
	descs <- nodeCPUUsageOnlineDesc
	descs <- nodeCPUUsageOfflineDesc
	descs <- nodeMemoryReservedDesc
	descs <- nodeMemoryUsageOnlineDesc
	descs <- nodeMemoryUsageOfflineDesc
}

func (n *NodeResourceCollector) CollectWithStability(metrics chan<- k8smetrics.Metric) {
	node, err := n.nodeLister.Get(n.nodeName)
	if err != nil {
		klog.ErrorS(err, "list pods failed")
		return
	}
	metrics <- k8smetrics.NewLazyConstMetric(nodeElasticCPUDesc, k8smetrics.GaugeValue, node.Status.Allocatable.Name(known.ElasticCPU, resource.DecimalSI).AsApproximateFloat64(), node.Name)
	metrics <- k8smetrics.NewLazyConstMetric(nodeElasticMemoryDesc, k8smetrics.GaugeValue, node.Status.Allocatable.Name(known.ElasticMemory, resource.BinarySI).AsApproximateFloat64(), node.Name)
	metrics <- k8smetrics.NewLazyConstMetric(nodeCPUAllocatableDesc, k8smetrics.GaugeValue, node.Status.Allocatable.Cpu().AsApproximateFloat64(), node.Name)
	metrics <- k8smetrics.NewLazyConstMetric(nodeMemoryAllocatableDesc, k8smetrics.GaugeValue, node.Status.Allocatable.Memory().AsApproximateFloat64(), node.Name)
	metrics <- k8smetrics.NewLazyConstMetric(nodeCPUCapacityDesc, k8smetrics.GaugeValue, node.Status.Capacity.Cpu().AsApproximateFloat64(), node.Name)
	metrics <- k8smetrics.NewLazyConstMetric(nodeMemoryCapacityDesc, k8smetrics.GaugeValue, node.Status.Capacity.Memory().AsApproximateFloat64(), node.Name)

	resourceStatus := n.nodeResourceGetter()
	if resourceStatus == nil {
		return
	}
	metrics <- k8smetrics.NewLazyConstMetric(nodeCPUReservedDesc, k8smetrics.GaugeValue, resourceStatus.CPUReserved.AsApproximateFloat64(), node.Name)
	// TODO incorrect online define !!
	metrics <- k8smetrics.NewLazyConstMetric(nodeCPUUsageOnlineDesc, k8smetrics.GaugeValue, resourceStatus.CPUUsage.AsApproximateFloat64(), node.Name)
	metrics <- k8smetrics.NewLazyConstMetric(nodeCPUUsageOfflineDesc, k8smetrics.GaugeValue, resourceStatus.CPUUsageOffline.AsApproximateFloat64(), node.Name)

	metrics <- k8smetrics.NewLazyConstMetric(nodeMemoryReservedDesc, k8smetrics.GaugeValue, resourceStatus.MemoryReserved.AsApproximateFloat64(), node.Name)
	// TODO incorrect online define !!
	metrics <- k8smetrics.NewLazyConstMetric(nodeMemoryUsageOnlineDesc, k8smetrics.GaugeValue, resourceStatus.MemoryUsage.AsApproximateFloat64(), node.Name)
	metrics <- k8smetrics.NewLazyConstMetric(nodeMemoryUsageOfflineDesc, k8smetrics.GaugeValue, resourceStatus.MemoryUsageOffline.AsApproximateFloat64(), node.Name)
}
