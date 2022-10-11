package types

type CollectType string

const (
	NodeLocalCollectorType            CollectType = "node-local"
	CadvisorCollectorType             CollectType = "cadvisor"
	EbpfCollectorType                 CollectType = "ebpf"
	MetricsServerCollectorType        CollectType = "metrics-server"
	NodeResourceCollectorType         CollectType = "node-resource"
	NodeResourceTopologyCollectorType CollectType = "node-resource-topology"
)

type MetricName string

const (
	MetricNameCpuTotalUsage       MetricName = "cpu_total_usage"
	MetricNameCpuTotalUtilization MetricName = "cpu_total_utilization"
	MetricNameCpuLoad1Min         MetricName = "cpu_load_1_min"
	MetricNameCpuLoad5Min         MetricName = "cpu_load_5_min"
	MetricNameCpuLoad15Min        MetricName = "cpu_load_15_min"
	MetricNameCpuCoreNumbers      MetricName = "cpu_core_numbers"

	MetricNameExclusiveCPUIdle MetricName = "exclusive_cpu_idle"

	MetricNameMemoryTotalUsage       MetricName = "memory_total_usage"
	MetricNameMemoryTotalUtilization MetricName = "memory_total_utilization"

	MetricDiskReadKiBPS   MetricName = "disk_read_kibps"
	MetricDiskWriteKiBPS  MetricName = "disk_write_kibps"
	MetricDiskReadIOPS    MetricName = "disk_read_iops"
	MetricDiskWriteIOPS   MetricName = "disk_read_iops"
	MetricDiskUtilization MetricName = "disk_read_utilization"

	MetricNetworkReceiveKiBPS MetricName = "network_receive_kibps"
	MetricNetworkSentKiBPS    MetricName = "network_sent_kibps"
	MetricNetworkReceivePckPS MetricName = "network_receive_pckps"
	MetricNetworkSentPckPS    MetricName = "network_sent_pckps"
	MetricNetworkDropIn       MetricName = "network_drop_in"
	MetricNetworkDropOut      MetricName = "network_drop_out"

	// Attention: this value is cpuUsageIncrease/timeIncrease, not cpuUsage
	MetricNameContainerCpuTotalUsage     MetricName = "container_cpu_total_usage"
	MetricNameContainerCpuLimit          MetricName = "container_cpu_limit"
	MetricNameContainerCpuQuota          MetricName = "container_cpu_quota"
	MetricNameContainerCpuPeriod         MetricName = "container_cpu_period"
	MetricNameContainerSchedRunQueueTime MetricName = "container_sched_run_queue_time"

	MetricNameExtResContainerCpuTotalUsage MetricName = "ext_res_container_cpu_total_usage"
	MetricNameExtCpuTotalDistribute        MetricName = "ext_cpu_total_distribute"

	MetricNameContainerMemTotalUsage       MetricName = "container_mem_total_usage"
	MetricNameExtResContainerMemTotalUsage MetricName = "ext_res_container_mem_total_usage"
)
