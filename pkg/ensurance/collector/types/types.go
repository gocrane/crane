package types

import (
	"strings"

	v1 "k8s.io/api/core/v1"

	"github.com/gocrane/crane/pkg/utils"
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
	NodeLocalCollectorType CollectType = "node-local"
)

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
	var pathArrays = []string{utils.CgroupKubePods}

	switch c.PodQOSClass {
	case v1.PodQOSGuaranteed:
		pathArrays = append(pathArrays, utils.CgroupPodPrefix+c.PodUid)
	case v1.PodQOSBurstable:
		pathArrays = append(pathArrays, strings.ToLower(string(v1.PodQOSBurstable)), utils.CgroupPodPrefix+c.PodUid)
	case v1.PodQOSBestEffort:
		pathArrays = append(pathArrays, strings.ToLower(string(v1.PodQOSBestEffort)), utils.CgroupPodPrefix+c.PodUid)
	default:
		return ""
	}

	if c.ContainerId != "" {
		pathArrays = append(pathArrays, c.ContainerId)
	}

	return strings.Join(pathArrays, "/")
}
