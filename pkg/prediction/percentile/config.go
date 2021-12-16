package percentile

import (
	"reflect"
	"sync"
	"time"

	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/recommender/util"

	"github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/utils"
)

//var metricToInternalConfigMap map[string]*internalConfig = map[string]*internalConfig{}
var queryToInternalConfigMap map[string]*internalConfig = map[string]*internalConfig{}

var mu = sync.Mutex{}

var defaultMinSampleWeight float64 = 1e-5
var defaultMarginFraction float64 = .15
var defaultPercentile float64 = .95
var defaultHistogramOptions, _ = vpa.NewLinearHistogramOptions(100.0, 0.1, 1e-10)

var defaultInternalConfig = internalConfig{
	sampleInterval:         time.Minute,
	histogramDecayHalfLife: time.Hour * 24,
	minSampleWeight:        defaultMinSampleWeight,
	marginFraction:         defaultMarginFraction,
	percentile:             defaultPercentile,
	histogramOptions:       defaultHistogramOptions,
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
}

func makeInternalConfig(p *v1alpha1.Percentile) (*internalConfig, error) {
	sampleInterval, err := utils.ParseDuration(p.SampleInterval)
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

	return &internalConfig{
		sampleInterval:         sampleInterval,
		histogramOptions:       options,
		histogramDecayHalfLife: halfLife,
		minSampleWeight:        minSampleWeight,
		marginFraction:         marginFraction,
		percentile:             percentile,
	}, nil
}

func getInternalConfig(queryExpr string) *internalConfig {
	mu.Lock()
	defer mu.Unlock()

	config, exits := queryToInternalConfigMap[queryExpr]
	if !exits {
		logger.Info("Percentile internal config not found, using the default one.", "queryExpr", queryExpr)
		queryToInternalConfigMap[queryExpr] = &defaultInternalConfig
		return queryToInternalConfigMap[queryExpr]
	}

	return config
}

var configUpdateEventReceiver config.Receiver
var configDeleteEventReceiver config.Receiver

func init() {
	configUpdateEventReceiver = config.UpdateEventBroadcaster.Listen()
	configDeleteEventReceiver = config.DeleteEventBroadcaster.Listen()

	go func() {
		for {
			cfg := configUpdateEventReceiver.Read().(*config.Config)
			if cfg.Percentile == nil {
				continue
			}

			internalCfg, err := makeInternalConfig(cfg.Percentile)
			if err != nil {
				logger.Error(err, "Failed to create interval config.")
				continue
			}

			mu.Lock()
			if cfg.Expression != nil && len(cfg.Expression.Expression) > 0 {
				orig, exists := queryToInternalConfigMap[cfg.Expression.Expression]
				if !exists || !reflect.DeepEqual(orig, internalCfg) {
					queryToInternalConfigMap[cfg.Expression.Expression] = internalCfg
				}
			}
			mu.Unlock()
		}
	}()

	go func() {
		for {
			cfg := configDeleteEventReceiver.Read().(*config.Config)

			mu.Lock()
			if cfg.Expression != nil {
				delete(queryToInternalConfigMap, cfg.Expression.Expression)
			}
			mu.Unlock()
		}
	}()
}
