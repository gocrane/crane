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
)

func init() {
	metrics.Registry.MustRegister(ResourceRecommendation)
}
