package timeseriesprediction

import (
	predictionv1alpha1 "github.com/gocrane/api/prediction/v1alpha1"
	"testing"
)

func TestPredictionMetricsEqualWithDefaultMetric(t *testing.T) {
	workload := workloadInfo{
		Namespace:  "default",
		Name:       "test-workload",
		APIVersion: "apps/v1",
		Kind:       "Deployment",
	}
	metric1 := getDefaultPredictionMetric(workload)
	metric2 := getDefaultPredictionMetric(workload)

	metricsA := []predictionv1alpha1.PredictionMetric{metric1}
	metricsB := []predictionv1alpha1.PredictionMetric{metric2}

	if !predictionMetricsEqual(metricsA, metricsB) {
		t.Errorf("predictionMetricsEqual should return true for identical metrics")
	}

	metric2.ExpressionQuery.Expression = "xxx"
	if predictionMetricsEqual(metricsA, metricsB) {
		t.Errorf("predictionMetricsEqual should return false for different metrics")
	}
}
