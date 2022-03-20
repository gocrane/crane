package dsp

import (
	"fmt"
	"time"

	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/prediction"
	"github.com/gocrane/crane/pkg/prediction/config"
)

// makeInternalConfig
// queryHistoryTimeSeries
// isPeriodicTimeSeries
//

// How to get a MetricContext ?
// func NewMetricContext(fetcher target.SelectorFetcher, seriesPrediction *predictionapi.TimeSeriesPrediction, predictorMgr predictormgr.Manager)
// c, err := NewMetricContext(tc.TargetFetcher, tsPrediction, tc.predictorMgr)


//scaleClient := scale.New(
//	discoveryClientSet.RESTClient(), mgr.GetRESTMapper(),
//	dynamic.LegacyAPIPathResolverFunc,
//	scaleKindResolver,
//)
//
//targetSelectorFetcher := target.NewSelectorFetcher(mgr.GetScheme(), mgr.GetRESTMapper(), scaleClient, mgr.GetClient())


func Debug(predictor *prediction.GenericPrediction, namer metricnaming.MetricNamer, config *config.Config) (*Signal, *Signal, *Signal, error) {
	internalConfig, err := makeInternalConfig(config.DSP)
	if err != nil {
		return nil, nil, nil, err
	}
klog.Infof("WWWW internalConfig:%v", *internalConfig)
	historyTimeSeriesList, err := queryHistoryTimeSeries(predictor, namer, internalConfig)
	if err != nil {
		return nil, nil, nil, err
	}

	queryExpr := namer.BuildUniqueKey()

	var signal, history, test, estimate *Signal
	var nCycles int
	var chosenEstimator Estimator
	for _, ts := range historyTimeSeriesList {
		if isPeriodicTimeSeries(ts, internalConfig.historyResolution, Day) {
			signal = SamplesToSignal(ts.Samples, internalConfig.historyResolution)
			signal, nCycles = signal.Truncate(Day)
			if nCycles >= 2 {
				chosenEstimator = bestEstimator(queryExpr, internalConfig.estimators, signal, nCycles, Day)
			}
			if chosenEstimator != nil {
				samplesPerCycle := len(signal.Samples) / nCycles
				history = &Signal{
					SampleRate: signal.SampleRate,
					Samples:    signal.Samples[:(nCycles-1)*samplesPerCycle],
				}
				test = &Signal{
					SampleRate: signal.SampleRate,
					Samples:    signal.Samples[(nCycles-1)*samplesPerCycle:],
				}
				estimate = chosenEstimator.GetEstimation(history, Day)
				return history, test, estimate, nil
			}
		}
	}

	return nil, nil, nil, fmt.Errorf("no prediction result")
}

func queryHistoryTimeSeries(predictor *prediction.GenericPrediction, namer metricnaming.MetricNamer, config *internalConfig) ([]*common.TimeSeries, error) {
	p := predictor.GetHistoryProvider()
	if p == nil {
		return nil, fmt.Errorf("history provider not provisioned")
	}
klog.Infof("WWWW history: %v", p)
	end := time.Now().Truncate(config.historyResolution)
	start := end.Add(-config.historyDuration - time.Hour)
klog.Infof("WWWW start: %v, end:%v", start, end)
	//tsList, err := p.QueryTimeSeries(namer, start, end, config.historyResolution)
	//if err != nil {
	//	klog.ErrorS(err, "Failed to query history time series.")
	//	return nil, err
	//}
	//
	//klog.V(6).InfoS("DSP debug | queryHistoryTimeSeries", "timeSeriesList", tsList, "config", *config)
	//
	//return preProcessTimeSeriesList(tsList, config)
	return []*common.TimeSeries{}, fmt.Errorf("dummy error")
}
