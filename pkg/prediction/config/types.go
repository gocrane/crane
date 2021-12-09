package config

import (
	"github.com/gocrane/api/prediction/v1alpha1"
)

type Config struct {
	MetricSelector *v1alpha1.MetricSelector
	Query          *v1alpha1.Query
	DSP            *v1alpha1.DSP
	Percentile     *v1alpha1.Percentile
}

// ConvertApiMetrics2InternalConfigs
func ConvertApiMetrics2InternalConfigs(metrics []v1alpha1.PredictionMetric) []*Config {
	var confs []*Config
	for _, metric := range metrics {
		confs = append(confs, ConvertApiMetric2InternalConfig(&metric))
	}
	return confs
}

// ConvertApiMetric2InternalConfig
func ConvertApiMetric2InternalConfig(metric *v1alpha1.PredictionMetric) *Config {
	// transfer the workload to query
	if metric.WorkloadResource != nil {
		// todo: different data source has different querys.
		query := &v1alpha1.Query{
			Expression: WorkloadResourceToPromQueryExpr(metric.WorkloadResource),
		}
		return &Config{
			Query:      query,
			DSP:        metric.Algorithm.DSP,
			Percentile: metric.Algorithm.Percentile,
		}
	} else if metric.NodeResource != nil {
		// todo: different data source has different querys.
		query := &v1alpha1.Query{
			Expression: NodeResourceToPromQueryExpr(metric.NodeResource),
		}
		return &Config{
			Query:      query,
			DSP:        metric.Algorithm.DSP,
			Percentile: metric.Algorithm.Percentile,
		}
	} else {
		return &Config{
			MetricSelector: metric.MetricSelector,
			Query:          metric.Query,
			DSP:            metric.Algorithm.DSP,
			Percentile:     metric.Algorithm.Percentile,
		}
	}
}
