package types

import (
	"strings"

	v1 "k8s.io/api/core/v1"
)

type CollectType string

type MetricName string

const (
	MetricNameCpuTotalUsage              MetricName = "cpu_total_usage"
	MetricNameCpuTotalUtilization        MetricName = "cpu_total_utilization"
	MetricNameContainerCpuTotalUsage     MetricName = "container_cpu_total_usage"
	MetricNameContainerCpuLimit          MetricName = "container_cpu_limit"
	MetricNameContainerCpuQuota          MetricName = "container_cpu_quota"
	MetricNameContainerCpuPeriod         MetricName = "container_cpu_period"
	MetricNameContainerSchedRunQueueTime MetricName = "container_sched_run_queue_time"
)

const (
	CgroupKubePods  = "/kubepods"
	CgroupPodPrefix = "pod"
)

const (
	NodeLocalCollectorType CollectType = "node-local"
	CadvisorCollectorType  CollectType = "cadvisor"
)

type MetricNameConfig struct {
	//metricName string
	//selector   metav1.LabelSelector
}

type MetricNameConfigs []MetricNameConfig

type UpdateEvent struct {
	Index uint64
}

// CgroupRef group pod infos
type CgroupRef struct {
	ContainerName string
	ContainerId   string
	PodName       string
	PodNamespace  string
	PodUid        string
	PodQOSClass   v1.PodQOSClass
}

func (c *CgroupRef) GetCgroupPath() string {
	var pathArrays = []string{CgroupKubePods}

	switch c.PodQOSClass {
	case v1.PodQOSGuaranteed:
		pathArrays = append(pathArrays, CgroupPodPrefix+c.PodUid)
	case v1.PodQOSBurstable:
		pathArrays = append(pathArrays, strings.ToLower(string(v1.PodQOSBurstable)), CgroupPodPrefix+c.PodUid)
	case v1.PodQOSBestEffort:
		pathArrays = append(pathArrays, strings.ToLower(string(v1.PodQOSBestEffort)), CgroupPodPrefix+c.PodUid)
	default:
		return ""
	}

	if c.ContainerId != "" {
		pathArrays = append(pathArrays, c.ContainerId)
	}

	return strings.Join(pathArrays, "/")
}
