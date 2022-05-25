package percentile

import (
	"fmt"
	"time"

	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/recommender/util"
	"k8s.io/klog/v2"

	"github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/utils"
)

var defaultMinSampleWeight float64 = 1e-5
var defaultMarginFraction float64 = 0.0
var defaultPercentile float64 = .99
var defaultTargetUtilization float64 = 1.0
var defaultHistogramOptions, _ = vpa.NewLinearHistogramOptions(100.0, 0.1, 1e-10)

var defaultInternalConfig = internalConfig{
	aggregated:             true,
	sampleInterval:         time.Minute,
	histogramDecayHalfLife: time.Hour * 24,
	minSampleWeight:        defaultMinSampleWeight,
	marginFraction:         defaultMarginFraction,
	percentile:             defaultPercentile,
	histogramOptions:       defaultHistogramOptions,
	targetUtilization:      defaultTargetUtilization,
	historyLength:          time.Hour * 24 * 7,
}

type internalConfig struct {
	aggregated             bool
	historyLength          time.Duration
	sampleInterval         time.Duration
	histogramOptions       vpa.HistogramOptions
	histogramDecayHalfLife time.Duration
	minSampleWeight        float64
	marginFraction         float64
	percentile             float64
	targetUtilization      float64
	initMode               config.ModelInitMode
}

func (c *internalConfig) String() string {
	return fmt.Sprintf("{aggregated: %v, historyLength: %v, sampleInterval: %v, histogramDecayHalfLife: %v, minSampleWeight: %v, marginFraction: %v, percentile: %v, targetUtilization: %v}",
		c.aggregated, c.historyLength, c.sampleInterval, c.histogramDecayHalfLife, c.minSampleWeight, c.marginFraction, c.percentile, c.targetUtilization)
}

// todo: later better to refine the algorithm params to a map not a struct to get more extendability,
// if not, we add some param is very difficult because it will modify crane api
func makeInternalConfig(p *v1alpha1.Percentile, initMode *config.ModelInitMode) (*internalConfig, error) {
	sampleInterval, err := utils.ParseDuration(p.SampleInterval)
	if err != nil {
		return nil, err
	}
	historyLength, err := utils.ParseDuration(p.HistoryLength)
	if err != nil {
		return nil, err
	}

	halfLife, err := utils.ParseDuration(p.Histogram.HalfLife)
	if err != nil {
		return nil, err
	}

	var options vpa.HistogramOptions

	if len(p.Histogram.BucketSizeGrowthRatio) > 0 &&
		len(p.Histogram.FirstBucketSize) > 0 &&
		len(p.Histogram.MaxValue) > 0 {
		bucketSizeGrowthRatio, err := utils.ParseFloat(p.Histogram.BucketSizeGrowthRatio, 0)
		if err != nil {
			return nil, err
		}

		firstBucketSize, err := utils.ParseFloat(p.Histogram.FirstBucketSize, 0)
		if err != nil {
			return nil, err
		}

		maxValue, err := utils.ParseFloat(p.Histogram.MaxValue, 0)
		if err != nil {
			return nil, err
		}

		epsilon, err := utils.ParseFloat(p.Histogram.Epsilon, 1e-10)
		if err != nil {
			return nil, err
		}

		options, err = vpa.NewExponentialHistogramOptions(maxValue, firstBucketSize, 1.0+bucketSizeGrowthRatio, epsilon)
		if err != nil {
			return nil, err
		}
	} else if len(p.Histogram.BucketSize) > 0 && len(p.Histogram.MaxValue) > 0 {
		bucketSize, err := utils.ParseFloat(p.Histogram.BucketSize, 0)
		if err != nil {
			return nil, err
		}

		maxValue, err := utils.ParseFloat(p.Histogram.MaxValue, 0)
		if err != nil {
			return nil, err
		}

		epsilon, err := utils.ParseFloat(p.Histogram.Epsilon, 1e-10)
		if err != nil {
			return nil, err
		}

		options, err = vpa.NewLinearHistogramOptions(maxValue, bucketSize, epsilon)
		if err != nil {
			return nil, err
		}
	} else {
		options = defaultHistogramOptions
	}

	percentile, err := utils.ParseFloat(p.Percentile, defaultPercentile)
	if err != nil {
		return nil, err
	}

	marginFraction, err := utils.ParseFloat(p.MarginFraction, defaultMarginFraction)
	if err != nil {
		return nil, err
	}

	minSampleWeight, err := utils.ParseFloat(p.MinSampleWeight, defaultMinSampleWeight)
	if err != nil {
		return nil, err
	}

	targetUtilization, err := utils.ParseFloat(p.TargetUtilization, defaultTargetUtilization)
	if err != nil {
		return nil, err
	}

	// default use history
	mode := config.ModelInitModeHistory
	if initMode != nil {
		mode = *initMode
	}
	c := &internalConfig{
		initMode:               mode,
		aggregated:             p.Aggregated,
		historyLength:          historyLength,
		sampleInterval:         sampleInterval,
		histogramOptions:       options,
		histogramDecayHalfLife: halfLife,
		minSampleWeight:        minSampleWeight,
		marginFraction:         marginFraction,
		percentile:             percentile,
		targetUtilization:      targetUtilization,
	}
	klog.InfoS("Made an internal config.", "internalConfig", c)

	return c, nil
}
