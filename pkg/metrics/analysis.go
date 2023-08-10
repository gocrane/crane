package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	ResourceRecommendation = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "crane",
			Subsystem: "analysis",
			Name:      "resource_recommendation",
			Help:      "The containers' CPU/Memory recommended value",
		},
		[]string{"apiversion", "owner_kind", "namespace", "owner_name", "container", "resource"},
	)

	ReplicasRecommendation = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "crane",
			Subsystem: "analysis",
			Name:      "replicas_recommendation",
			Help:      "The workload's replicas recommended value",
		},
		[]string{"apiversion", "owner_kind", "namespace", "owner_name"},
	)

	SelectTargets = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "crane",
			Subsystem: "analysis",
			Name:      "select_targets",
			Help:      "The number of selected targets",
		},
		[]string{"type", "apiversion", "owner_kind", "namespace", "owner_name"},
	)

	RecommendationsStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "crane",
			Subsystem: "analysis",
			Name:      "recommendations_status",
			Help:      "The status of recommendations",
		},
		[]string{"type", "apiversion", "owner_kind", "namespace", "owner_name", "update_status", "result_status"},
	)
)

func init() {
	metrics.Registry.MustRegister(ResourceRecommendation, ReplicasRecommendation, SelectTargets, RecommendationsStatus)
}
