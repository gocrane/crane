package percentile

import (
	"github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/utils"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/recommender/util"
	"reflect"
	"strconv"

	"sync"
	"time"

	"github.com/gocrane/crane/pkg/prediction/config"
)

//var metricToInternalConfigMap map[string]*internalConfig = map[string]*internalConfig{}
var queryToInternalConfigMap map[string]*internalConfig = map[string]*internalConfig{}

var mu = sync.Mutex{}

var withMetricEventBroadcaster config.Broadcaster = config.NewBroadcaster()

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
		bucketSizeGrowthRatio, err := parseFloat(p.Histogram.BucketSizeGrowthRatio, 0)
		if err != nil {
			return nil, err
		}
		
		firstBucketSize, err := parseFloat(p.Histogram.FirstBucketSize, 0)
		if err != nil {
			return nil, err
		}
		
		maxValue, err := parseFloat(p.Histogram.MaxValue, 0)
		if err != nil {
			return nil, err
		}
		
		epsilon, err := parseFloat(p.Histogram.Epsilon, 1e-10)
		if err != nil {
			return nil, err
		}
		
		options, err = vpa.NewExponentialHistogramOptions(maxValue, firstBucketSize, 1.0+bucketSizeGrowthRatio, epsilon)
		if err != nil {
			return nil, err
		}
	} else if len(p.Histogram.BucketSize) > 0 && len(p.Histogram.MaxValue) > 0 {
		bucketSize, err := parseFloat(p.Histogram.BucketSize, 0) 
		if err != nil {
			return nil, err
		}
		
		maxValue, err := parseFloat(p.Histogram.MaxValue, 0)
		if err != nil {
			return nil, err
		}

		epsilon, err := parseFloat(p.Histogram.Epsilon, 1e-10)
		if err != nil {
			return nil, err
		}
		options, err = vpa.NewLinearHistogramOptions(maxValue, bucketSize, epsilon)
	} else {
		options = defaultHistogramOptions
	}

	
	percentile, err := parseFloat(p.Percentile, defaultPercentile)
	if err != nil {
		return nil, err
	}

	marginFraction, err := parseFloat(p.MarginFraction, defaultMarginFraction)
	if err != nil {
		return nil, err
	}

	minSampleWeight, err := parseFloat(p.MinSampleWeight, defaultMinSampleWeight)
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

func parseFloat(str string, defaultValue float64) (float64, error) {
	if len(str) == 0 {
		return defaultValue, nil
	}
	return strconv.ParseFloat(str, 64)
}

func getInternalConfig(queryExpr string) *internalConfig {
	mu.Lock()
	defer mu.Unlock()

	config, exits := queryToInternalConfigMap[queryExpr]
	if !exits {
		logger.Info("Internal config not found, using the default one.", "queryExpr", queryExpr)
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
			if cfg.Query != nil && len(cfg.Query.Expression) > 0 {
				orig, exists := queryToInternalConfigMap[cfg.Query.Expression]
				if !exists || !reflect.DeepEqual(orig, internalCfg) {
					queryToInternalConfigMap[cfg.Query.Expression] = internalCfg
				}
			}
			mu.Unlock()
		}
	}()

	go func() {
		for {
			cfg := configDeleteEventReceiver.Read().(config.Config)

			mu.Lock()
			if cfg.Query != nil {
				delete(queryToInternalConfigMap, cfg.Query.Expression)
			}
			mu.Unlock()
		}
	}()
}
