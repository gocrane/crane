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

func Debug(predictor prediction.Interface, namer metricnaming.MetricNamer, config *config.Config) (*Signal, *Signal, *Signal, error) {
	internalConfig, err := makeInternalConfig(config.DSP)
	if err != nil {
		return nil, nil, nil, err
	}

	historyTimeSeriesList, err := queryHistoryTimeSeries(predictor.(*periodicSignalPrediction), namer, internalConfig)
	if err != nil {
		return nil, nil, nil, err
	}

	queryExpr := namer.BuildUniqueKey()

	var signal, history, test, estimate *Signal
	var nPeriods int
	var chosenEstimator Estimator
	for _, ts := range historyTimeSeriesList {
		periodLength := findPeriod(ts, internalConfig.historyResolution)
		if periodLength == Day || periodLength == Week {
			signal = SamplesToSignal(ts.Samples, internalConfig.historyResolution)
			signal, nPeriods = signal.Truncate(periodLength)
			if nPeriods >= 2 {
				chosenEstimator = bestEstimator(queryExpr, internalConfig.estimators, signal, nPeriods, periodLength)
			}
			if chosenEstimator != nil {
				samplesPerPeriod := len(signal.Samples) / nPeriods
				history = &Signal{
					SampleRate: signal.SampleRate,
					Samples:    signal.Samples[:(nPeriods-1)*samplesPerPeriod],
				}
				test = &Signal{
					SampleRate: signal.SampleRate,
					Samples:    signal.Samples[(nPeriods-1)*samplesPerPeriod:],
				}
				estimate = chosenEstimator.GetEstimation(history, periodLength)
				return history, test, estimate, nil
			}
		}
	}

	return nil, nil, nil, fmt.Errorf("no prediction result")
}

func queryHistoryTimeSeries(predictor *periodicSignalPrediction, namer metricnaming.MetricNamer, config *internalConfig) ([]*common.TimeSeries, error) {
	p := predictor.GetHistoryProvider()
	if p == nil {
		return nil, fmt.Errorf("history provider not provisioned")
	}

	end := time.Now().Truncate(config.historyResolution)
	start := end.Add(-config.historyDuration - time.Hour)

	tsList, err := p.QueryTimeSeries(namer, start, end, config.historyResolution)
	if err != nil {
		klog.ErrorS(err, "Failed to query history time series.")
		return nil, err
	}

	klog.V(4).InfoS("DSP debug | queryHistoryTimeSeries", "timeSeriesList", tsList, "config", *config)

	return preProcessTimeSeriesList(tsList, config)
}
