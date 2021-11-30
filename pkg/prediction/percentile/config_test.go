package percentile

import (
	"github.com/stretchr/testify/assert"
	"testing"
	//"time"

)

var expr = "container_cpu_core_used"

//var cfg *config.Config = &config.Config{
//	MetricName: &metricName,
//	Percentile: &config.PercentileConfig{
//		SampleInterval:  "15s",
//		MinSampleWeight: 0.01,
//		Histogram: config.HistogramConfig{
//			MaxValue:   100,
//			HalfLife:   "12h",
//			BucketSize: 10,
//			Epsilon:    1e-15,
//		},
//	},
//}

func TestConfig(t *testing.T) {
	internalConfig := getInternalConfig(expr)
	assert.Equal(t, &defaultInternalConfig, internalConfig)

	//// Add a config
	//config.WithConfig(cfg)
	//
	//// Wait a second for the internal config being added
	//time.Sleep(time.Second)
	//
	//// Verify that the config has been added
	//ic, exists = metricToInternalConfigMap[metricName]
	//assert.True(t, exists)
	//assert.NotNil(t, ic)
	//assert.Equal(t, time.Second*15, ic.SampleInterval)
	//assert.Equal(t, 0.01, ic.MinSampleWeight)
	//assert.Equal(t, time.Hour*12, ic.HistogramDecayHalfLife)
	//assert.Equal(t, 11, ic.HistogramOptions.NumBuckets())
	//assert.Equal(t, 1e-15, ic.HistogramOptions.Epsilon())
	//
	//// Delete a config
	//config.DeleteConfig(*cfg)
	//
	//// Wait a second for the internal config being deleted
	//time.Sleep(time.Second)
	//
	//// Verify that the config has been removed
	//ic, exists = metricToInternalConfigMap[metricName]
	//assert.False(t, exists)
	//assert.Nil(t, ic)
}
