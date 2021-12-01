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