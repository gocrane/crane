package utils

import (
	"fmt"
)

// todo: later we change these templates to configurable like prometheus-adapter
const (
	// WorkloadCpuUsageExprTemplate is used to query workload cpu usage by promql,  param is namespace,workload-name,duration str
	WorkloadCpuUsageExprTemplate = `sum(irate(container_cpu_usage_seconds_total{namespace="%s",pod=~"^%s-.*$",container!=""}[%s]))`
	// WorkloadMemUsageExprTemplate is used to query workload mem usage by promql, param is namespace, workload-name
	WorkloadMemUsageExprTemplate = `sum(container_memory_working_set_bytes{namespace="%s",pod=~"^%s-.*$",container!=""})`

	// following is node exporter metric for node cpu/memory usage
	// NodeCpuUsageExprTemplate is used to query node cpu usage by promql,  param is node name which prometheus scrape, duration str
	NodeCpuUsageExprTemplate = `sum(count(node_cpu_seconds_total{mode="idle",instance=~"(%s)(:\\d+)?"}) by (mode, cpu)) - sum(irate(node_cpu_seconds_total{mode="idle",instance=~"(%s)(:\\d+)?"}[%s]))`
	// NodeMemUsageExprTemplate is used to query node cpu memory by promql,  param is node name, node name which prometheus scrape
	NodeMemUsageExprTemplate = `sum(node_memory_MemTotal_bytes{instance=~"(%s)(:\\d+)?"} - node_memory_MemAvailable_bytes{instance=~"(%s)(:\\d+)?"})`

	// PodCpuUsageExprTemplate is used to query pod cpu usage by promql,  param is namespace,pod, duration str
	PodCpuUsageExprTemplate = `sum(irate(container_cpu_usage_seconds_total{container!="POD",namespace="%s",pod="%s"}[%s]))`
	// PodMemUsageExprTemplate is used to query pod cpu usage by promql,  param is namespace,pod
	PodMemUsageExprTemplate = `sum(container_memory_working_set_bytes{container!="POD",namespace="%s",pod="%s"})`

	// ContainerCpuUsageExprTemplate is used to query container cpu usage by promql,  param is namespace,pod,container duration str
	ContainerCpuUsageExprTemplate = `irate(container_cpu_usage_seconds_total{container!="POD",namespace="%s",pod=~"^%s.*$",container="%s"}[%s])`
	// ContainerMemUsageExprTemplate is used to query container cpu usage by promql,  param is namespace,pod,container
	ContainerMemUsageExprTemplate = `container_memory_working_set_bytes{container!="POD",namespace="%s",pod=~"^%s.*$",container="%s"}`

	CustomerExprTemplate = `sum(%s{%s})`
)

func GetCustomerExpression(metricName string, labels string) string {
	return fmt.Sprintf(CustomerExprTemplate, metricName, labels)
}

func GetWorkloadCpuUsageExpression(namespace string, name string) string {
	return fmt.Sprintf(WorkloadCpuUsageExprTemplate, namespace, name, "3m")
}

func GetWorkloadMemUsageExpression(namespace string, name string) string {
	return fmt.Sprintf(WorkloadMemUsageExprTemplate, namespace, name)
}

func GetContainerCpuUsageExpression(namespace string, workloadName string, containerName string) string {
	return fmt.Sprintf(ContainerCpuUsageExprTemplate, namespace, workloadName, containerName, "3m")
}

func GetContainerMemUsageExpression(namespace string, workloadName string, containerName string) string {
	return fmt.Sprintf(ContainerMemUsageExprTemplate, namespace, workloadName, containerName)
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
