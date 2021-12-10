package types

type CollectType string

type MetricName string

const (
	MetricNamCpuTotalUsage       MetricName = "cpu_total_usage"
	MetricNamCpuTotalUtilization MetricName = "cpu_total_utilization"
)

const (
	NodeLocalCollectorType CollectType = "node-local"
)

type MetricNameConfig struct {
	//metricName string
	//selector   metav1.LabelSelector
}

type MetricNameConfigs []MetricNameConfig

type UpdateEvent struct {
	Index uint64
}
