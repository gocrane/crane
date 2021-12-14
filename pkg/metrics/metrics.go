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

	TSPMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "crane",
			Subsystem: "prediction",
			Name:      "time_series_prediction_metric",
			Help:      "prediction value for TimeSeriesPrediction",
		},
		[]string{"targetKind", "targetName", "targetNamespace", "resourceIdentifier", "type", "resourceQuery", "expressionQuery", "rawQuery", "algorithm", "aggregateKey"},
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(HPAReplicas, EHPAReplicas)
}
