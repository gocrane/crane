package known

const (
	CraneSystemNamespace = "crane-system"
)

const (
	MetricNamePodCpuUsage = "crane_pod_cpu_usage"
)

const (
	DefaultCoolDownSeconds            = 300
	DefaultRestoredThreshold          = 1
	DefaultAvoidedThreshold           = 1
	DefaultDeletionGracePeriodSeconds = 30
	MaxMinCPURatio                    = 100
	MaxStepCPURatio                   = 100
)
