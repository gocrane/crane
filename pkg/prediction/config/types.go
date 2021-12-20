package config

import (
	"time"

	"github.com/gocrane/api/prediction/v1alpha1"
)

type AlgorithmModelConfig struct {
	UpdateInterval time.Duration
}

type Config struct {
	Metric     *v1alpha1.MetricQuery
	Expression *v1alpha1.ExpressionQuery
	DSP        *v1alpha1.DSP
	Percentile *v1alpha1.Percentile
}

// ConvertApiMetrics2InternalConfigs
func (c *MetricContext) ConvertApiMetrics2InternalConfigs(metrics []v1alpha1.PredictionMetric) []*Config {
	var confs []*Config
	for _, metric := range metrics {
		confs = append(confs, c.ConvertApiMetric2InternalConfig(&metric))
	}
	return confs
}

// ConvertApiMetric2InternalConfig
func (c *MetricContext) ConvertApiMetric2InternalConfig(metric *v1alpha1.PredictionMetric) *Config {
	// transfer the workload to query
	if metric.ResourceQuery != nil {
		// todo: different data source has different querys.
		expr := &v1alpha1.ExpressionQuery{
			Expression: c.ResourceToPromQueryExpr(metric.ResourceQuery),
		}
		return &Config{
			Expression: expr,
			DSP:        metric.Algorithm.DSP,
			Percentile: metric.Algorithm.Percentile,
		}
	} else {
		return &Config{
			Metric:     metric.MetricQuery,
			Expression: metric.ExpressionQuery,
			DSP:        metric.Algorithm.DSP,
			Percentile: metric.Algorithm.Percentile,
		}
	}
}
