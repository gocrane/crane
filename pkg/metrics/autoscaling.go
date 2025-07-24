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
		[]string{"namespace", "name"},
	)
	EHPAReplicas = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "crane",
			Subsystem: "autoscaling",
			Name:      "effective_hpa_replicas",
			Help:      "Replicas for Effective HPA",
		},
		[]string{"namespace", "name"},
	)
	HPAScaleCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "crane",
			Subsystem: "autoscaling",
			Name:      "hpa_scale_count",
			Help:      "Scale count for HPA",
		},
		[]string{"namespace", "name", "type"},
	)
	OOMCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "crane",
			Subsystem: "autoscaling",
			Name:      "oom_count",
			Help:      "The count of pod oom event",
		},
		[]string{
			"namespace",
			"pod",
			"container",
		},
	)
	EVPACpuScaleUp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "crane",
			Subsystem: "autoscaling",
			Name:      "effective_vpa_cpu_scale_up",
			Help:      "The cpu scale up for Effective VPA",
		},
		[]string{"apiversion", "owner_kind", "namespace", "owner_name", "container", "resource"},
	)
	EVPACpuScaleDown = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "crane",
			Subsystem: "autoscaling",
			Name:      "effective_vpa_cpu_scale_down",
			Help:      "The cpu scale down for Effective VPA",
		},
		[]string{"apiversion", "owner_kind", "namespace", "owner_name", "container", "resource"},
	)
	EVPAMemoryScaleUp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "crane",
			Subsystem: "autoscaling",
			Name:      "effective_vpa_memory_scale_up",
			Help:      "The memory scale up for Effective VPA",
		},
		[]string{"apiversion", "owner_kind", "namespace", "owner_name", "container", "resource"},
	)
	EVPAMemoryScaleDown = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "crane",
			Subsystem: "autoscaling",
			Name:      "effective_vpa_memory_scale_down",
			Help:      "The memory scale down for Effective VPA",
		},
		[]string{"apiversion", "owner_kind", "namespace", "owner_name", "container", "resource"},
	)
	EVPAResourceRecommendation = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "crane",
			Subsystem: "autoscaling",
			Name:      "effective_vpa_resource_recommendation",
			Help:      "The resource recommendation for Effective VPA",
		},
		[]string{"apiversion", "owner_kind", "namespace", "owner_name", "container", "resource"},
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(HPAReplicas, EHPAReplicas, OOMCount, HPAScaleCount, EVPACpuScaleUp, EVPACpuScaleDown, EVPAMemoryScaleDown, EVPAMemoryScaleUp, EVPAResourceRecommendation)

}

func CustomCollectorRegister(collector ...prometheus.Collector) {
	metrics.Registry.MustRegister(collector...)
}
