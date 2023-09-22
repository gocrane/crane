package utils

import (
	"fmt"
	"strings"
)

// todo: later we change these templates to configurable like prometheus-adapter
const (
	ExtensionLabelsHolder = `EXTENSION_LABELS_HOLDER`
	// WorkloadCpuUsageExprTemplate is used to query workload cpu usage by promql,  param is namespace,workload-name,duration str
	WorkloadCpuUsageExprTemplate = `sum(irate(container_cpu_usage_seconds_total{namespace="%s",pod=~"%s",container!=""EXTENSION_LABELS_HOLDER}[%s]))`
	// WorkloadMemUsageExprTemplate is used to query workload mem usage by promql, param is namespace, workload-name
	WorkloadMemUsageExprTemplate = `sum(container_memory_working_set_bytes{namespace="%s",pod=~"%s",container!=""EXTENSION_LABELS_HOLDER})`

	// following is node exporter metric for node cpu/memory usage
	// NodeCpuUsageExprTemplate is used to query node cpu usage by promql,  param is node name which prometheus scrape, duration str
	NodeCpuUsageExprTemplate = `sum(count(node_cpu_seconds_total{mode="idle",instance=~"(%s)(:\\d+)?"EXTENSION_LABELS_HOLDER}) by (mode, cpu)) - sum(irate(node_cpu_seconds_total{mode="idle",instance=~"(%s)(:\\d+)?"EXTENSION_LABELS_HOLDER}[%s]))`
	// NodeMemUsageExprTemplate is used to query node memory usage by promql,  param is node name, node name which prometheus scrape
	NodeMemUsageExprTemplate = `sum(node_memory_MemTotal_bytes{instance=~"(%s)(:\\d+)?EXTENSION_LABELS_HOLDER"} - node_memory_MemAvailable_bytes{instance=~"(%s)(:\\d+)?"EXTENSION_LABELS_HOLDER})`

	// NodeCpuRequestUtilizationExprTemplate is used to query node cpu request utilization by promql, param is node name, node name which prometheus scrape
	NodeCpuRequestUtilizationExprTemplate = `sum(kube_pod_container_resource_requests{node="%s", resource="cpu", unit="core"EXTENSION_LABELS_HOLDER} * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"EXTENSION_LABELS_HOLDER}) by (node)) by (node) / sum(kube_node_status_capacity{node="%s", resource="cpu", unit="core"EXTENSION_LABELS_HOLDER} * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"EXTENSION_LABELS_HOLDER}) by (node)) by (node) `
	// NodeMemRequestUtilizationExprTemplate is used to query node memory request utilization by promql, param is node name, node name which prometheus scrape
	NodeMemRequestUtilizationExprTemplate = `sum(kube_pod_container_resource_requests{node="%s", resource="memory", unit="byte", namespace!=""EXTENSION_LABELS_HOLDER} * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"EXTENSION_LABELS_HOLDER}) by (node)) by (node) / sum(kube_node_status_capacity{node="%s", resource="memory", unit="byte"EXTENSION_LABELS_HOLDER} * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"EXTENSION_LABELS_HOLDER}) by (node)) by (node) `
	// NodeCpuUsageUtilizationExprTemplate is used to query node memory usage utilization by promql, param is node name, node name which prometheus scrape
	NodeCpuUsageUtilizationExprTemplate = `sum(label_replace(irate(container_cpu_usage_seconds_total{instance="%s", container!="POD", container!="",image!=""EXTENSION_LABELS_HOLDER}[1h]), "node", "$1", "instance",  "(^[^:]+)") * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"EXTENSION_LABELS_HOLDER}) by (node)) by (node) / sum(kube_node_status_capacity{node="%s", resource="cpu", unit="core"EXTENSION_LABELS_HOLDER} * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"EXTENSION_LABELS_HOLDER}) by (node)) by (node) `
	// NodeMemUsageUtilizationExprTemplate is used to query node memory usage utilization by promql, param is node name, node name which prometheus scrape
	NodeMemUsageUtilizationExprTemplate = `sum(label_replace(container_memory_usage_bytes{instance="%s", namespace!="",container!="POD", container!="",image!=""EXTENSION_LABELS_HOLDER}, "node", "$1", "instance", "(^[^:]+)") * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"EXTENSION_LABELS_HOLDER}) by (node)) by (node) / sum(kube_node_status_capacity{node="%s", resource="memory", unit="byte"EXTENSION_LABELS_HOLDER} * on (node) group_left() max(kube_node_labels{label_beta_kubernetes_io_instance_type!~"eklet", label_node_kubernetes_io_instance_type!~"eklet"EXTENSION_LABELS_HOLDER}) by (node)) by (node) `

	// PodCpuUsageExprTemplate is used to query pod cpu usage by promql,  param is namespace,pod, duration str
	PodCpuUsageExprTemplate = `sum(irate(container_cpu_usage_seconds_total{container!="POD",namespace="%s",pod="%s"EXTENSION_LABELS_HOLDER}[%s]))`
	// PodMemUsageExprTemplate is used to query pod cpu usage by promql,  param is namespace,pod
	PodMemUsageExprTemplate = `sum(container_memory_working_set_bytes{container!="POD",namespace="%s",pod="%s"EXTENSION_LABELS_HOLDER})`

	// ContainerCpuUsageExprTemplate is used to query container cpu usage by promql,  param is namespace,pod,container duration str
	ContainerCpuUsageExprTemplate = `irate(container_cpu_usage_seconds_total{container!="POD",namespace="%s",pod=~"%s",container="%s"EXTENSION_LABELS_HOLDER}[%s])`
	// ContainerMemUsageExprTemplate is used to query container cpu usage by promql,  param is namespace,pod,container
	ContainerMemUsageExprTemplate = `container_memory_working_set_bytes{container!="POD",namespace="%s",pod=~"%s",container="%s"EXTENSION_LABELS_HOLDER}`

	CustomerExprTemplate = `sum(%s{%sEXTENSION_LABELS_HOLDER})`

	// Container network cumulative count of bytes received
	queryFmtNetReceiveBytes = `sum(rate(container_network_receive_bytes_total{namespace="%s",pod=~"%s",container!=""EXTENSION_LABELS_HOLDER}[3m]))`
	// Container network cumulative count of bytes transmitted
	queryFmtNetTransferBytes = `sum(rate(container_network_transmit_bytes_total{namespace="%s",pod=~"%s",container!=""EXTENSION_LABELS_HOLDER}[3m]))`
)

const (
	PostRegMatchesPodDeployment  = `[a-z0-9]+-[a-z0-9]{5}$`
	PostRegMatchesPodReplicaset  = `[a-z0-9]+$`
	PostRegMatchesPodDaemonSet   = `[a-z0-9]{5}$`
	PostRegMatchesPodStatefulset = `[0-9]+$`
)

var ExtensionLabelArray []string
var extensionLabelsString string

func SetExtensionLabels(extensionLabels string) {
	if extensionLabels != "" {
		for _, label := range strings.Split(extensionLabels, ",") {
			ExtensionLabelArray = append(ExtensionLabelArray, label)
		}

		extensionLabelsString = ","
		for index, label := range ExtensionLabelArray {
			labelArr := strings.Split(label, "=")
			if len(labelArr) != 2 {
				// skip the invalid kv
				continue
			}

			extensionLabelsString += fmt.Sprintf("%s=\"%s\"", labelArr[0], labelArr[1])
			if index != len(ExtensionLabelArray)-1 {
				extensionLabelsString += ","
			}
		}
	}
}

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
	return fmtSprintfInternal(CustomerExprTemplate, metricName, labels)
}

func GetWorkloadCpuUsageExpression(namespace string, name string, kind string) string {
	return fmtSprintfInternal(WorkloadCpuUsageExprTemplate, namespace, GetPodNameReg(name, kind), "3m")
}

func GetWorkloadMemUsageExpression(namespace string, name string, kind string) string {
	return fmtSprintfInternal(WorkloadMemUsageExprTemplate, namespace, GetPodNameReg(name, kind))
}

func GetContainerCpuUsageExpression(namespace string, workloadName string, kind string, containerName string) string {
	return fmtSprintfInternal(ContainerCpuUsageExprTemplate, namespace, GetPodNameReg(workloadName, kind), containerName, "3m")
}

func GetContainerMemUsageExpression(namespace string, workloadName string, kind string, containerName string) string {
	return fmtSprintfInternal(ContainerMemUsageExprTemplate, namespace, GetPodNameReg(workloadName, kind), containerName)
}

func GetPodCpuUsageExpression(namespace string, name string) string {
	return fmtSprintfInternal(PodCpuUsageExprTemplate, namespace, name, "3m")
}

func GetPodMemUsageExpression(namespace string, name string) string {
	return fmtSprintfInternal(PodMemUsageExprTemplate, namespace, name)
}

func GetNodeCpuUsageExpression(nodeName string) string {
	return fmtSprintfInternal(NodeCpuUsageExprTemplate, nodeName, nodeName, "3m")
}

func GetNodeMemUsageExpression(nodeName string) string {
	return fmtSprintfInternal(NodeMemUsageExprTemplate, nodeName, nodeName)
}

func GetNodeCpuRequestUtilizationExpression(nodeName string) string {
	return fmtSprintfInternal(NodeCpuRequestUtilizationExprTemplate, nodeName, nodeName)
}

func GetNodeMemRequestUtilizationExpression(nodeName string) string {
	return fmtSprintfInternal(NodeMemRequestUtilizationExprTemplate, nodeName, nodeName)
}

func GetNodeCpuUsageUtilizationExpression(nodeName string) string {
	return fmtSprintfInternal(NodeCpuUsageUtilizationExprTemplate, nodeName, nodeName)
}

func GetNodeMemUsageUtilizationExpression(nodeName string) string {
	return fmtSprintfInternal(NodeMemUsageUtilizationExprTemplate, nodeName, nodeName)
}

func GetWorkloadNetReceiveBytesExpression(namespace string, name string, kind string) string {
	return fmtSprintfInternal(queryFmtNetReceiveBytes, namespace, GetPodNameReg(name, kind))
}

func GetWorkloadNetTransferBytesExpression(namespace string, name string, kind string) string {
	return fmtSprintfInternal(queryFmtNetTransferBytes, namespace, GetPodNameReg(name, kind))
}

func fmtSprintfInternal(format string, a ...interface{}) string {
	formatReplaced := strings.ReplaceAll(format, ExtensionLabelsHolder, extensionLabelsString)
	return fmt.Sprintf(formatReplaced, a...)
}
