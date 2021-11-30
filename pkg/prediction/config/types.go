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

//type DspConfig struct {
//	// SampleInterval is the sampling interval of metrics.
//	SampleInterval string
//	// HistoryLength describes how long back should be queried against provider to get historical metrics for prediction.
//	HistoryLength string
//	//
//	Estimators *EstimatorConfigs
//}
//
//type EstimatorConfigs struct {
//	MaxValue []*MaxValueEstimatorConfig
//	FFT      []*FFTEstimatorConfig
//}
//
//type MaxValueEstimatorConfig struct{}
//
//type FFTEstimatorConfig struct {
//	MarginFraction         float64
//	LowAmplitudeThreshold  float64
//	HighFrequencyThreshold float64
//	MinNumOfSpectrumItems  int
//	MaxNumOfSpectrumItems  int
//}
//
//type PercentileConfig struct {
//	SampleInterval  string
//	Histogram       HistogramConfig
//	MinSampleWeight *float64
//	MarginFraction  *float64
//	Percentile      *float64
//}
//
//type HistogramConfig struct {
//	MaxValue              float64
//	Epsilon               float64
//	HalfLife              string
//	BucketSize            float64
//	FirstBucketSize       float64
//	BucketSizeGrowthRatio float64
//}
