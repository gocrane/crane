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
