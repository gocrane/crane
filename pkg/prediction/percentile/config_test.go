package percentile

import (
	"testing"
	"time"

	"github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/stretchr/testify/assert"
)

var expr = "container_cpu_core_used"

var cfg *config.Config = &config.Config{
	Expression: &v1alpha1.ExpressionQuery{Expression: expr},
	Percentile: &v1alpha1.Percentile{
		SampleInterval:  "15s",
		MinSampleWeight: "0.01",
		Histogram: v1alpha1.HistogramConfig{
			MaxValue:   "100",
			HalfLife:   "12h",
			BucketSize: "10",
			Epsilon:    "1e-15",
		},
	},
}

func TestConfig(t *testing.T) {
	internalConfig := getInternalConfig(expr)
	assert.Equal(t, &defaultInternalConfig, internalConfig)
	c := &config.MetricContext{}
	// Add a config
	c.WithConfig(cfg)

	// Wait a second for the internal config being added
	time.Sleep(time.Second)
	//
	// Verify that the config has been added
	internalCfg := getInternalConfig(expr)
	assert.NotNil(t, internalCfg)
	assert.Equal(t, time.Second*15, internalCfg.sampleInterval)
	assert.Equal(t, 0.01, internalCfg.minSampleWeight)
	assert.Equal(t, time.Hour*12, internalCfg.histogramDecayHalfLife)
	assert.Equal(t, 11, internalCfg.histogramOptions.NumBuckets())
	assert.Equal(t, 1e-15, internalCfg.histogramOptions.Epsilon())

	// Delete a config
	c.DeleteConfig(cfg)

	// Wait a second for the internal config being deleted
	time.Sleep(time.Second)

	// Verify that the config has been removed
	internalCfg = getInternalConfig(expr)
	assert.Equal(t, &defaultInternalConfig, internalConfig)
}
