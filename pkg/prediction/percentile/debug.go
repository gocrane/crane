package percentile

import (
	"fmt"
	"time"

	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/prediction"
	"github.com/gocrane/crane/pkg/prediction/config"
)

func Debug(predictor prediction.Interface, namer metricnaming.MetricNamer, config *config.Config) ([]*common.TimeSeries, []*common.TimeSeries, error) {
	cfg, err := makeInternalConfig(config.Percentile, config.InitMode)
	if err != nil {
		return nil, nil, err
	}

	p := predictor.(*percentilePrediction)
	historyTimeSeriesList, err := queryHistoryTimeSeries(p, namer, cfg)
	if err != nil {
		return nil, nil, err
	}

	queryExpr := namer.BuildUniqueKey()
	klog.V(4).Infof("process analyzing metric namer: %v, config: %+v", namer.BuildUniqueKey(), *cfg)

	signals := map[string]*aggregateSignal{}
	if cfg.aggregated {
		signal := newAggregateSignal(cfg)
		for _, ts := range historyTimeSeriesList {
			for _, s := range ts.Samples {
				t := time.Unix(s.Timestamp, 0)
				signal.addSample(t, s.Value)
			}
		}
		signals[keyAll] = signal
	} else {
		for _, ts := range historyTimeSeriesList {
			if len(ts.Samples) < 1 {
				continue
			}
			key := prediction.AggregateSignalKey(ts.Labels)
			signal := newAggregateSignal(cfg)
			for _, s := range ts.Samples {
				t := time.Unix(s.Timestamp, 0)
				signal.addSample(t, s.Value)
			}
			signal.labels = ts.Labels
			signals[key] = signal
		}
	}

	return historyTimeSeriesList, p.getPredictedValuesFromSignals(queryExpr, signals, cfg), nil

}

func queryHistoryTimeSeries(predictor *percentilePrediction, namer metricnaming.MetricNamer, config *internalConfig) ([]*common.TimeSeries, error) {
	p := predictor.GetHistoryProvider()
	if p == nil {
		return nil, fmt.Errorf("history provider not provisioned")
	}

	end := time.Now().Truncate(time.Minute)
	start := end.Add(-config.historyLength)
	historyTimeSeries, err := predictor.GetHistoryProvider().QueryTimeSeries(namer, start, end, config.sampleInterval)
	if err != nil {
		klog.ErrorS(err, "Failed to query history time series.")
		return nil, err
	}
	return historyTimeSeries, nil
}
