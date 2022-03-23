package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	HPAReplicas = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "crane",
			Subsystem: "autoscaling",
			Name:      "hpa_replicas",
			Help:      "Replicas for HPA",
		},
		[]string{"identity"},
	)
	EHPAReplicas = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "crane",
			Subsystem: "autoscaling",
			Name:      "effective_hpa_replicas",
			Help:      "Replicas for Effective HPA",
		},
		[]string{"identity", "strategy"},
	)
	OOMCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "crane",
			Subsystem: "autoscaling",
			Name:      "oom_count",
			Help:      "The count of pod oom event",
		},
		[]string{
			"pod",
			"container",
		},
	)
	EVPACpuScaleUpMilliCores = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "crane",
			Subsystem: "autoscaling",
			Name:      "effective_vpa_cpu_scale_up_millicores",
			Help:      "The cpu scale up for Effective VPA",
		},
		[]string{
			"target",
		},
	)
	EVPACpuScaleDownMilliCores = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "crane",
			Subsystem: "autoscaling",
			Name:      "effective_vpa_cpu_scale_down_millicores",
			Help:      "The cpu scale down for Effective VPA",
		},
		[]string{
			"target",
		},
	)
	EVPAMemoryScaleUpMB = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "crane",
			Subsystem: "autoscaling",
			Name:      "effective_vpa_memory_scale_up_mb",
			Help:      "The memory scale up for Effective VPA",
		},
		[]string{
			"target",
		},
	)
	EVPAMemoryScaleDownMB = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "crane",
			Subsystem: "autoscaling",
			Name:      "effective_vpa_memory_scale_down_mb",
			Help:      "The memory scale down for Effective VPA",
		},
		[]string{
			"target",
		},
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(HPAReplicas, EHPAReplicas, OOMCount, EVPACpuScaleUpMilliCores, EVPACpuScaleDownMilliCores, EVPAMemoryScaleDownMB, EVPAMemoryScaleUpMB)

}

func CustomCollectorRegister(collector ...prometheus.Collector) {
	metrics.Registry.MustRegister(collector...)
}
