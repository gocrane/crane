package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	k8smetrics "k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
	"k8s.io/klog/v2"
)

// This const block defines the metric names for the crane-agent metrics.
const (
	CraneNamespace      = "crane"
	CraneAgentSubsystem = "craneAgent"

	LastActivity                = "last_activity"
	StepDurationSeconds         = "step_duration_seconds"
	StepDurationQuantileSummary = "step_duration_quantile_summary"

	AnalyzerStatus        = "analyzer_status"
	AnalyzerStatusTotal   = "analyzer_status_total"
	ExecutorStatus        = "executor_status"
	ExecutorStatusTotal   = "executor_status_total"
	ExecutorErrorTotal    = "executor_error_total"
	ExecutorEvictTotal    = "executor_evict_total"
	PodResourceErrorTotal = "pod_resource_error_total"

	NodeCpuCannotBeReclaimedSeconds = "node_cpu_cannot_be_reclaimed_seconds"
	NodeResourceRecommended         = "node_resource_recommended"
	NodeResourceRecommendedFrom     = "node_resource_recommended_from"
)

type StepLabel string

const (
	StepMain               StepLabel = "main"
	StepCollect            StepLabel = "collect"
	StepAvoid              StepLabel = "avoid"
	StepRestore            StepLabel = "restore"
	StepUpdateConfig       StepLabel = "updateConfig"
	StepUpdateNodeResource StepLabel = "updateNodeResource"
	StepUpdatePodResource  StepLabel = "updatePodResource"

	// Step for pod resource manager
	StepGetPeriod   StepLabel = "getPeriod"
	StepUpdateQuota StepLabel = "updateQuota"

	StepGetExtResourceRecommended StepLabel = "getExtResourceRecommended"
)

type SubComponent string

const (
	SubComponentSchedule     SubComponent = "schedule"
	SubComponentThrottle     SubComponent = "throttle"
	SubComponentEvict        SubComponent = "evict"
	SubComponentPodResource  SubComponent = "pod-resource-manager"
	SubComponentNodeResource SubComponent = "node-resource-manager"
)

type AnalyzeType string

const (
	AnalyzeTypeEnableScheduling AnalyzeType = "enableScheduling"
	AnalyzeTypeAvoidance        AnalyzeType = "avoidance"
	AnalyzeTypeRestore          AnalyzeType = "restore"
	AnalyzeTypeAnalyzeError     AnalyzeType = "analyzeError "
)

const (
	// LogLongDurationThreshold defines the duration after which long step
	// duration will be logged (in addition to being counted in metric).
	// This is meant to help find unexpectedly long step execution times for
	// debugging purposes.
	LogLongDurationThreshold = 1 * time.Minute
)

var (

	// LastActivity records the last activity time of each steps
	lastActivity = k8smetrics.NewGaugeVec(
		&k8smetrics.GaugeOpts{
			Namespace:      CraneNamespace,
			Subsystem:      CraneAgentSubsystem,
			Name:           LastActivity,
			Help:           "Last time certain part of crane logic executed.",
			StabilityLevel: k8smetrics.ALPHA,
		}, []string{"module", "subcomponent", "step"},
	)

	//StepDuration records the time cost of each steps
	stepDuration = k8smetrics.NewHistogramVec(
		&k8smetrics.HistogramOpts{
			Namespace:      CraneNamespace,
			Subsystem:      CraneAgentSubsystem,
			Name:           StepDurationSeconds,
			Help:           "Time taken by various steps of the crane-agent.",
			Buckets:        []float64{0.01, 0.05, 0.1, 0.5, 1.0, 2.5, 5.0, 7.5, 10.0, 12.5, 15.0, 17.5, 20.0, 22.5, 25.0, 27.5, 30.0, 50.0, 75.0, 100.0, 1000.0},
			StabilityLevel: k8smetrics.ALPHA,
		}, []string{"module", "subcomponent", "step"},
	)

	stepDurationSummary = k8smetrics.NewSummaryVec(
		&k8smetrics.SummaryOpts{
			Namespace:      CraneNamespace,
			Subsystem:      CraneAgentSubsystem,
			Name:           StepDurationQuantileSummary,
			Help:           "Quantiles of time taken by various steps of the crane-agent.",
			MaxAge:         time.Hour,
			StabilityLevel: k8smetrics.ALPHA,
		}, []string{"module", "subcomponent", "step"},
	)

	//AnalyzerStatus records the status of analyzer module
	analyzerStatus = k8smetrics.NewGaugeVec(
		&k8smetrics.GaugeOpts{
			Namespace:      CraneNamespace,
			Subsystem:      CraneAgentSubsystem,
			Name:           AnalyzerStatus,
			Help:           "Status of anomaly analyzer.",
			StabilityLevel: k8smetrics.ALPHA,
		}, []string{"key", "type"},
	)

	analyzerStatusCounts = k8smetrics.NewCounterVec(
		&k8smetrics.CounterOpts{
			Namespace:      CraneNamespace,
			Subsystem:      CraneAgentSubsystem,
			Name:           AnalyzerStatusTotal,
			Help:           "The times of NodeQOS rule triggered/restored.",
			StabilityLevel: k8smetrics.ALPHA,
		}, []string{"key", "type"},
	)

	//ExecutorStatus records the status of executor module
	executorStatus = k8smetrics.NewGaugeVec(
		&k8smetrics.GaugeOpts{
			Namespace:      CraneNamespace,
			Subsystem:      CraneAgentSubsystem,
			Name:           ExecutorStatus,
			Help:           "Status of action executor.",
			StabilityLevel: k8smetrics.ALPHA,
		}, []string{"subcomponent", "step"},
	)

	executorStatusCounts = k8smetrics.NewCounterVec(
		&k8smetrics.CounterOpts{
			Namespace:      CraneNamespace,
			Subsystem:      CraneAgentSubsystem,
			Name:           ExecutorStatusTotal,
			Help:           "The times of action executor triggered/restored.",
			StabilityLevel: k8smetrics.ALPHA,
		}, []string{"subcomponent", "step"},
	)

	//ExecutorErrorCounts records the number of errors when execute actions
	executorErrorCounts = k8smetrics.NewCounterVec(
		&k8smetrics.CounterOpts{
			Namespace:      CraneNamespace,
			Subsystem:      CraneAgentSubsystem,
			Name:           ExecutorErrorTotal,
			Help:           "The error times of action executor triggered/restored.",
			StabilityLevel: k8smetrics.ALPHA,
		}, []string{"subcomponent", "step"},
	)

	//ExecutorEvictCounts records the number of pods evicted by executor module
	executorEvictCounts = k8smetrics.NewCounter(
		&k8smetrics.CounterOpts{
			Namespace:      CraneNamespace,
			Subsystem:      CraneAgentSubsystem,
			Name:           ExecutorEvictTotal,
			Help:           "The number of evicted pods.",
			StabilityLevel: k8smetrics.ALPHA,
		},
	)

	//podResourceUpdateErrorCounts records the number of errors when update pod's ext resource to quota
	podResourceUpdateErrorCounts = k8smetrics.NewCounterVec(
		&k8smetrics.CounterOpts{
			Namespace:      CraneNamespace,
			Subsystem:      CraneAgentSubsystem,
			Name:           PodResourceErrorTotal,
			Help:           "The error times for pod resource manager to update.",
			StabilityLevel: k8smetrics.ALPHA,
		}, []string{"subcomponent", "step"},
	)

	// LastActivity records the last activity time of each steps
	nodeCpuCannotBeReclaimedSeconds = k8smetrics.NewGaugeVec(
		&k8smetrics.GaugeOpts{
			Namespace:      CraneNamespace,
			Subsystem:      CraneAgentSubsystem,
			Name:           NodeCpuCannotBeReclaimedSeconds,
			Help:           "The cpu seconds that cannot be reclaimed.",
			StabilityLevel: k8smetrics.ALPHA,
		}, []string{},
	)

	//NodeResourceRecommended
	nodeResourceRecommended = k8smetrics.NewGaugeVec(
		&k8smetrics.GaugeOpts{
			Namespace:      CraneNamespace,
			Subsystem:      CraneAgentSubsystem,
			Name:           NodeResourceRecommended,
			Help:           "The value of recommendation.",
			StabilityLevel: k8smetrics.ALPHA,
		}, []string{"subcomponent", "step", "resourceName"},
	)

	//NodeResourceRecommended
	nodeResourceRecommendedFrom = k8smetrics.NewGaugeVec(
		&k8smetrics.GaugeOpts{
			Namespace:      CraneNamespace,
			Subsystem:      CraneAgentSubsystem,
			Name:           NodeResourceRecommendedFrom,
			Help:           "Where the recommended values come from. (tsp: 1, local: 0)",
			StabilityLevel: k8smetrics.ALPHA,
		}, []string{"subcomponent", "step", "resourceName"},
	)
)

var registerCraneAgentMetricsOnce sync.Once

func RegisterCraneAgent() {
	registerCraneAgentMetricsOnce.Do(func() {
		legacyregistry.MustRegister(lastActivity)
		legacyregistry.MustRegister(stepDuration)
		legacyregistry.MustRegister(stepDurationSummary)
		legacyregistry.MustRegister(analyzerStatus)
		legacyregistry.MustRegister(analyzerStatusCounts)
		legacyregistry.MustRegister(executorStatus)
		legacyregistry.MustRegister(executorStatusCounts)
		legacyregistry.MustRegister(executorErrorCounts)
		legacyregistry.MustRegister(executorEvictCounts)
		legacyregistry.MustRegister(nodeCpuCannotBeReclaimedSeconds)
		legacyregistry.MustRegister(nodeResourceRecommended)
		legacyregistry.MustRegister(nodeResourceRecommendedFrom)
	})
}

// UpdateDurationFromStart records the duration of the step identified by the
// label using start time
func UpdateDurationFromStart(module string, stepName StepLabel, start time.Time) {
	duration := time.Since(start)
	UpdateDuration(module, stepName, duration)
}

func UpdateDurationFromStartWithSubComponent(module string, subComponent string, stepName StepLabel, start time.Time) {
	duration := time.Since(start)
	UpdateDurationWithSubComponent(module, subComponent, stepName, duration)
}

func UpdateDuration(module string, stepName StepLabel, duration time.Duration) {
	UpdateDurationWithSubComponent(module, "", stepName, duration)
}

func UpdateDurationWithSubComponent(module string, subComponent string, stepName StepLabel, duration time.Duration) {
	if duration > LogLongDurationThreshold {
		klog.V(4).Infof("Module %s, step %s took %v to complete", module, stepName, duration)
	}

	stepDuration.With(prometheus.Labels{"module": module, "subcomponent": subComponent, "step": string(stepName)}).Observe(duration.Seconds())
	stepDurationSummary.With(prometheus.Labels{"module": module, "subcomponent": subComponent, "step": string(stepName)}).Observe(duration.Seconds())
}

func UpdateLastTime(module string, stepName StepLabel, now time.Time) {
	UpdateLastTimeWithSubComponent(module, "", stepName, now)
}

func UpdateLastTimeWithSubComponent(module string, subComponent string, stepName StepLabel, now time.Time) {
	lastActivity.With(prometheus.Labels{"module": module, "subcomponent": subComponent, "step": string(stepName)}).Set(float64(now.Unix()))
}

func UpdateExecutorStatus(subComponent SubComponent, stepName StepLabel, value float64) {
	executorStatus.With(prometheus.Labels{"subcomponent": string(subComponent), "step": string(stepName)}).Set(value)
}

func UpdateAnalyzerStatus(typeName AnalyzeType, value float64) {
	analyzerStatus.With(prometheus.Labels{"type": string(typeName), "key": ""}).Set(value)
}

func UpdateAnalyzerWithKeyStatus(typeName AnalyzeType, key string, value float64) {
	analyzerStatus.With(prometheus.Labels{"type": string(typeName), "key": key}).Set(value)
}

func ExecutorStatusCounterInc(subComponent SubComponent, stepName StepLabel) {
	executorStatusCounts.With(prometheus.Labels{"subcomponent": string(subComponent), "step": string(stepName)}).Inc()
}

func ExecutorErrorCounterInc(subComponent SubComponent, stepName StepLabel) {
	executorErrorCounts.With(prometheus.Labels{"subcomponent": string(subComponent), "step": string(stepName)}).Inc()
}

func PodResourceUpdateErrorCounterInc(subComponent SubComponent, stepName StepLabel) {
	podResourceUpdateErrorCounts.With(prometheus.Labels{"subcomponent": string(subComponent), "step": string(stepName)}).Inc()
}

func ExecutorEvictCountsInc() {
	executorEvictCounts.Inc()
}

func UpdateNodeCpuCannotBeReclaimedSeconds(value float64) {
	nodeCpuCannotBeReclaimedSeconds.With(prometheus.Labels{}).Set(value)
}

func UpdateNodeResourceRecommendedValue(subComponent SubComponent, stepName StepLabel, resourceName string, from string, value float64) {
	nodeResourceRecommended.With(prometheus.Labels{"subcomponent": string(subComponent), "step": string(stepName), "resourceName": resourceName}).Set(value)
	switch from {
	case "tsp":
		UpdateNodeResourceRecommendedFromValue(subComponent, stepName, resourceName, 1)
	case "local":
		UpdateNodeResourceRecommendedFromValue(subComponent, stepName, resourceName, 0)
	default:
		UpdateNodeResourceRecommendedFromValue(subComponent, stepName, resourceName, -1)
	}
}

func UpdateNodeResourceRecommendedFromValue(subComponent SubComponent, stepName StepLabel, resourceName string, value float64) {
	nodeResourceRecommendedFrom.With(prometheus.Labels{"subcomponent": string(subComponent), "step": string(stepName), "resourceName": resourceName}).Set(value)
}
