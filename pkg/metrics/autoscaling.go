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
		[]string{"resourceName"},
	)
	EHPAReplicas = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "crane",
			Subsystem: "autoscaling",
			Name:      "effective_hpa_replicas",
			Help:      "Replicas for Effective HPA",
		},
		[]string{"resourceName", "strategy"},
	)
	HPAScaleCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "crane",
			Subsystem: "autoscaling",
			Name:      "hpa_scale_count",
			Help:      "Scale count for HPA",
		},
		[]string{"resourceName", "type", "direction"},
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
			"resourceName",
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
			"resourceName",
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
			"resourceName",
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
			"resourceName",
		},
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(HPAReplicas, EHPAReplicas, OOMCount, HPAScaleCount, EVPACpuScaleUpMilliCores, EVPACpuScaleDownMilliCores, EVPAMemoryScaleDownMB, EVPAMemoryScaleUpMB)

}

func CustomCollectorRegister(collector ...prometheus.Collector) {
	metrics.Registry.MustRegister(collector...)
}
