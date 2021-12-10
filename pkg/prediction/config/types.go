package config

import (
	"github.com/gocrane/api/prediction/v1alpha1"
)

type Config struct {
	MetricSelector *v1alpha1.ExpressionQuery
	Query          *v1alpha1.RawQuery
	DSP            *v1alpha1.DSP
	Percentile     *v1alpha1.Percentile
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
		query := &v1alpha1.RawQuery{
			Expression: c.ResourceToPromQueryExpr(metric.ResourceQuery),
		}
		return &Config{
			Query:      query,
			DSP:        metric.Algorithm.DSP,
			Percentile: metric.Algorithm.Percentile,
		}
	} else {
		return &Config{
			MetricSelector: metric.ExpressionQuery,
			Query:          metric.RawQuery,
			DSP:            metric.Algorithm.DSP,
			Percentile:     metric.Algorithm.Percentile,
		}
	}
}
