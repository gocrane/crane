package dsp

import (
	"github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var expr = "irate(container_cpu_core_used[3m])"

var cfg = &config.Config {
	Query: &v1alpha1.Query{Expression: expr},
	DSP: &v1alpha1.DSP {
		SampleInterval: "15s",
		HistoryLength:  "14d",
		Estimators: v1alpha1.Estimators{
			MaxValueEstimators: []*v1alpha1.MaxValueEstimator{{MarginFraction: "0.11"}, {MarginFraction: "0.09"}},
			FFTEstimators: []*v1alpha1.FFTEstimator{
				{MarginFraction: "0.05", LowAmplitudeThreshold: "0.9", HighFrequencyThreshold: "10."},
			},
		},
	},
}

func TestConfig(t *testing.T) {
	internalCfg := getInternalConfig(expr)
	assert.Equal(t, &defaultInternalConfig, internalCfg)

	// Add a config
	config.WithConfig(cfg)

	// Wait a second for the internal config being added
	time.Sleep(time.Second)

	// Verify that the config has been added
	internalCfg = getInternalConfig(expr)
	assert.NotNil(t, internalCfg)
	assert.Equal(t, time.Second*15, internalCfg.historyResolution)
	assert.Equal(t, time.Hour*24*14, internalCfg.historyDuration)

	assert.Equal(t, 3, len(internalCfg.estimators))
	assert.IsType(t, &maxValueEstimator{}, internalCfg.estimators[0])
	assert.IsType(t, &maxValueEstimator{}, internalCfg.estimators[1])
	assert.IsType(t, &fftEstimator{}, internalCfg.estimators[2])

	e0 := internalCfg.estimators[0].(*maxValueEstimator)
	assert.Equal(t, 0.11, e0.marginFraction)

	e1 := internalCfg.estimators[1].(*maxValueEstimator)
	assert.Equal(t, 0.09, e1.marginFraction)


	e2 := internalCfg.estimators[2].(*fftEstimator)
	assert.Equal(t, 0.05, e2.marginFraction)
	assert.Equal(t, 0.9, e2.lowAmplitudeThreshold)
	assert.Equal(t, 10., e2.highFrequencyThreshold)
	assert.Equal(t, defaultMinNumOfSpectrumItems, e2.minNumOfSpectrumItems)
	assert.Equal(t, defaultMaxNumOfSpectrumItems, e2.maxNumOfSpectrumItems)

	// Delete a config
	config.DeleteConfig(cfg)

	// Wait a second for the internal config being deleted
	time.Sleep(time.Second)

	// Verify that the config has been removed
	internalCfg = getInternalConfig(expr)
	assert.Equal(t, &defaultInternalConfig, internalCfg)
}
