package known

const (
	MetricNamePrediction  = "crane_autoscaling_prediction"
	MetricNameCron        = "crane_autoscaling_cron"
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
