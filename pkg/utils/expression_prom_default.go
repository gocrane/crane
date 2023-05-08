package utils

import (
	"fmt"
)

// todo: later we change these templates to configurable like prometheus-adapter
const (
	// WorkloadCpuUsageExprTemplate is used to query workload cpu usage by promql,  param is namespace,workload-name,duration str
	WorkloadCpuUsageExprTemplate = `sum(irate(container_cpu_usage_seconds_total{namespace="%s",pod=~"%s",container!=""}[%s]))`
	// WorkloadMemUsageExprTemplate is used to query workload mem usage by promql, param is namespace, workload-name
	WorkloadMemUsageExprTemplate = `sum(container_memory_working_set_bytes{namespace="%s",pod=~"%s",container!=""})`

	// following is node exporter metric for node cpu/memory usage
	// NodeCpuUsageExprTemplate is used to query node cpu usage by promql,  param is node name which prometheus scrape, duration str
	NodeCpuUsageExprTemplate = `sum(count(node_cpu_seconds_total{mode="idle",instance=~"(%s)(:\\d+)?"}) by (mode, cpu)) - sum(irate(node_cpu_seconds_total{mode="idle",instance=~"(%s)(:\\d+)?"}[%s]))`
	// NodeMemUsageExprTemplate is used to query node memory usage by promql,  param is node name, node name which prometheus scrape
	NodeMemUsageExprTemplate = `sum(node_memory_MemTotal_bytes{instance=~"(%s)(:\\d+)?"} - node_memory_MemAvailable_bytes{instance=~"(%s)(:\\d+)?"})`

	// NodeCpuRequestUtilizationExprTemplate is used to query node cpu request utilization by promql, param is node name, node name which prometheus scrape
	NodeCpuRequestUtilizationExprTemplate = `sum(kube_pod_container_resource_requests{node="%s", resource="cpu", unit="core"} * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)) by (node) / sum(kube_node_status_capacity{node="%s", resource="cpu", unit="core"} * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)) by (node) * 100`
	// NodeMemRequestUtilizationExprTemplate is used to query node memory request utilization by promql, param is node name, node name which prometheus scrape
	NodeMemRequestUtilizationExprTemplate = `sum(kube_pod_container_resource_requests{node="%s", resource="memory", unit="byte", namespace!=""} * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)) by (node) / sum(kube_node_status_capacity{node="%s", resource="memory", unit="byte"} * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)) by (node) * 100`
	// NodeCpuUsageUtilizationExprTemplate is used to query node memory usage utilization by promql, param is node name, node name which prometheus scrape
	NodeCpuUsageUtilizationExprTemplate = `sum(label_replace(irate(container_cpu_usage_seconds_total{instance="%s", container!="POD", container!="",image!=""}[1h]), "node", "$1", "instance",  "(^[^:]+)") * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)) by (node) / sum(kube_node_status_capacity{node="%s", resource="cpu", unit="core"} * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)) by (node) * 100`
	// NodeMemUsageUtilizationExprTemplate is used to query node memory usage utilization by promql, param is node name, node name which prometheus scrape
	NodeMemUsageUtilizationExprTemplate = `sum(label_replace(container_memory_usage_bytes{instance="%s", namespace!="",container!="POD", container!="",image!=""}, "node", "$1", "instance", "(^[^:]+)") * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)) by (node) / sum(kube_node_status_capacity{node="%s", resource="memory", unit="byte"} * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"}) by (node)) by (node) * 100`

	// PodCpuUsageExprTemplate is used to query pod cpu usage by promql,  param is namespace,pod, duration str
	PodCpuUsageExprTemplate = `sum(irate(container_cpu_usage_seconds_total{container!="POD",namespace="%s",pod="%s"}[%s]))`
	// PodMemUsageExprTemplate is used to query pod cpu usage by promql,  param is namespace,pod
	PodMemUsageExprTemplate = `sum(container_memory_working_set_bytes{container!="POD",namespace="%s",pod="%s"})`

	// ContainerCpuUsageExprTemplate is used to query container cpu usage by promql,  param is namespace,pod,container duration str
	ContainerCpuUsageExprTemplate = `irate(container_cpu_usage_seconds_total{container!="POD",namespace="%s",pod=~"%s",container="%s"}[%s])`
	// ContainerMemUsageExprTemplate is used to query container cpu usage by promql,  param is namespace,pod,container
	ContainerMemUsageExprTemplate = `container_memory_working_set_bytes{container!="POD",namespace="%s",pod=~"%s",container="%s"}`

	CustomerExprTemplate = `sum(%s{%s})`

	// Container network cumulative count of bytes received
	queryFmtNetReceiveBytes = `sum(increase(container_network_receive_bytes_total{pod=~"%s"}[1h])) by (namespace)`
	// Container network cumulative count of bytes transmitted
	queryFmtNetTransferBytes = `sum(increase(container_network_transmit_bytes_total{pod=~"%s"}[1h])) by (namespace)`
)

const (
	PostRegMatchesPodDeployment  = `[a-z0-9]+-[a-z0-9]{5}$`
	PostRegMatchesPodReplicaset  = `[a-z0-9]+$`
	PostRegMatchesPodDaemonSet   = `[a-z0-9]{5}$`
	PostRegMatchesPodStatefulset = `[0-9]+$`
)

func GetPodNameReg(resourceName string, resourceType string) string {
	switch resourceType {
	case "DaemonSet":
		return fmt.Sprintf("^%s-%s", resourceName, PostRegMatchesPodDaemonSet)
	case "ReplicaSet":
		return fmt.Sprintf("^%s-%s", resourceName, PostRegMatchesPodReplicaset)
	case "Deployment":
		return fmt.Sprintf("^%s-%s", resourceName, PostRegMatchesPodDeployment)
	case "StatefulSet":
		return fmt.Sprintf("^%s-%s", resourceName, PostRegMatchesPodStatefulset)
	}
	return fmt.Sprintf("^%s-%s", resourceName, `.*`)
}

func GetCustomerExpression(metricName string, labels string) string {
	return fmt.Sprintf(CustomerExprTemplate, metricName, labels)
}

func GetWorkloadCpuUsageExpression(namespace string, name string, kind string) string {
	return fmt.Sprintf(WorkloadCpuUsageExprTemplate, namespace, GetPodNameReg(name, kind), "3m")
}

func GetWorkloadMemUsageExpression(namespace string, name string, kind string) string {
	return fmt.Sprintf(WorkloadMemUsageExprTemplate, namespace, GetPodNameReg(name, kind))
}

func GetContainerCpuUsageExpression(namespace string, workloadName string, kind string, containerName string) string {
	return fmt.Sprintf(ContainerCpuUsageExprTemplate, namespace, GetPodNameReg(workloadName, kind), containerName, "3m")
}

func GetContainerMemUsageExpression(namespace string, workloadName string, kind string, containerName string) string {
	return fmt.Sprintf(ContainerMemUsageExprTemplate, namespace, GetPodNameReg(workloadName, kind), containerName)
}

func GetPodCpuUsageExpression(namespace string, name string) string {
	return fmt.Sprintf(PodCpuUsageExprTemplate, namespace, name, "3m")
}

func GetPodMemUsageExpression(namespace string, name string) string {
	return fmt.Sprintf(PodMemUsageExprTemplate, namespace, name)
}

func GetNodeCpuUsageExpression(nodeName string) string {
	return fmt.Sprintf(NodeCpuUsageExprTemplate, nodeName, nodeName, "3m")
}

func GetNodeMemUsageExpression(nodeName string) string {
	return fmt.Sprintf(NodeMemUsageExprTemplate, nodeName, nodeName)
}

func GetNodeCpuRequestUtilizationExpression(nodeName string) string {
	return fmt.Sprintf(NodeCpuRequestUtilizationExprTemplate, nodeName, nodeName)
}

func GetNodeMemRequestUtilizationExpression(nodeName string) string {
	return fmt.Sprintf(NodeMemRequestUtilizationExprTemplate, nodeName, nodeName)
}

func GetNodeCpuUsageUtilizationExpression(nodeName string) string {
	return fmt.Sprintf(NodeCpuUsageUtilizationExprTemplate, nodeName, nodeName)
}

func GetNodeMemUsageUtilizationExpression(nodeName string) string {
	return fmt.Sprintf(NodeMemUsageUtilizationExprTemplate, nodeName, nodeName)
}

func GetWorkloadNetReceiveBytesExpression(podName string) string {
	return fmt.Sprintf(queryFmtNetReceiveBytes, podName)
}

func GetWorkloadNetTransferBytesExpression(podName string) string {
	return fmt.Sprintf(queryFmtNetTransferBytes, podName)
}
