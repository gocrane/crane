package timeseriesprediction

import (
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func TestUpdateFromConfigMap(t *testing.T) {
	cm := &corev1.ConfigMap{
		Data: map[string]string{
			"Resource": `
predictionWindowSeconds: 86400
predictionMetrics:
- resourceIdentifier: workload-cpu
  type: ExpressionQuery
  expressionQuery:
    expression: "max(rate(container_cpu_usage_seconds_total{cluster=\"cls-6qvlzw6j\", namespace=\"%s\", pod=~\"%s\", container=\"\"}[5m]))"
  algorithm:
    algorithmType: sp
    sp:
      histogram:
        bucketSize: "0.01"
        epsilon: "1e-10"
        maxValue: "10.0"
        halfLife: "0h"
      minSampleWeight: "1.0"
      sampleInterval: "5m"
      historyLength: "24h"
`,
		},
	}

	store := NewTSPTemplateStore()
	store.UpdateFromConfigMap(cm)

	template, exists := store.GetTemplate("Resource")
	assert.True(t, exists, "模板应该被加载")
	assert.Equal(t, int32(86400), template.PredictionWindowSeconds)
	assert.Equal(t, 1, len(template.PredictionMetrics))
	assert.Equal(t, "workload-cpu", template.PredictionMetrics[0].ResourceIdentifier)
	assert.Equal(t, predictionapi.MetricType("ExpressionQuery"), template.PredictionMetrics[0].Type)
	assert.NotNil(t, template.PredictionMetrics[0].ExpressionQuery)
	assert.Equal(t, "max(rate(container_cpu_usage_seconds_total{cluster=\"cls-6qvlzw6j\", namespace=\"%s\", pod=~\"%s\", container=\"\"}[5m]))", template.PredictionMetrics[0].ExpressionQuery.Expression)
}
