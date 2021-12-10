package percentile

import (
	"fmt"
	"k8s.io/klog/v2"
	"time"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/prediction/config"

	"github.com/gocrane/crane/pkg/prediction"
)

var _ prediction.Interface = &percentilePrediction{}

type percentilePrediction struct {
	prediction.GenericPrediction
	a aggregateSignalMap
	//mr config.Receiver
	qr config.Receiver
}

//func (p *percentilePrediction) GetPredictedTimeSeries(metricName string,
//	conditions []common.QueryCondition, startTime time.Time, endTime time.Time) ([]*common.TimeSeries, error) {
//	return p.GetRealtimePredictedValues(metricName, conditions)
//}

func (p *percentilePrediction) QueryPredictedTimeSeries(rawQuery string, startTime time.Time, endTime time.Time) ([]*common.TimeSeries, error) {
	panic("Percentile algorithm cannot predict time series.")
}

func (p *percentilePrediction) QueryRealtimePredictedValues(queryExpr string) ([]*common.TimeSeries, error) {
	if p.GetRealtimeProvider() == nil {
		return nil, fmt.Errorf("realtime data provider not set")
	}

	cfg := getInternalConfig(queryExpr)

	estimator := NewPercentileEstimator(cfg.percentile)
	estimator = WithMargin(cfg.marginFraction, estimator)

	latestTimeSeries, err := p.GetRealtimeProvider().QueryLatestTimeSeries(queryExpr)
	if err != nil {
		return nil, err
	}
	logger.Info("Query latest time series", "latestTimeSeries", latestTimeSeries)

	if cfg.aggregated {
		key := prediction.AggregateSignalKey(queryExpr, nil)
		s, exists := p.a.Load(key)
		if
	}

	estimatedTimeSeries := make([]*common.TimeSeries, 0)

	now := time.Now().Unix()
	for _, ts := range latestTimeSeries {
		key := prediction.AggregateSignalKey(queryExpr, ts.Labels)
		s, exists := p.a.Load(key)
		if !exists {
			klog.Warningf("aggregate signal [%s] not found", key)
			continue
		}

		sample := common.Sample{
			Value:     estimator.GetEstimation(s.histogram),
			Timestamp: now,
		}

		estimatedTimeSeries = append(estimatedTimeSeries, &common.TimeSeries{
			Labels:  ts.Labels,
			Samples: []common.Sample{sample},
		})
	}

	return estimatedTimeSeries, nil
}

func NewPrediction() prediction.Interface {
	qb := config.NewBroadcaster()
	return &percentilePrediction{
		GenericPrediction: prediction.NewGenericPrediction(qb),
		a:                 aggregateSignalMap{},
		qr: qb.Listen(),
	}
}

func (p *percentilePrediction) Run(stopCh <-chan struct{}) {
	for {
		// Waiting for a new config
		cfg := p.qr.Read().(config.Config)

		go func(cfg config.Config) {
			expr := cfg.Query.Expression
			if expr == "" {
				return
			}
			if internalCfg := getInternalConfig(expr); internalCfg != nil {
				ticker := time.NewTicker(internalCfg.sampleInterval)

				defer ticker.Stop()
				for {
					p.addSamples(expr)
					select {
					case <-stopCh:
						return
					case <-ticker.C:
						continue
					}
				}
			}
		}(cfg)
	}
}

//func (p *percentilePrediction) GetRealtimePredictedValues(metricName string, conditions []common.QueryCondition) ([]*common.TimeSeries, error) {
//	if p.GetRealtimeProvider() == nil {
//		return nil, fmt.Errorf("realtime data provider not set")
//	}
//
//	cfg := getInternalConfigByMetricName(metricName)
//
//	estimator := NewPercentileEstimator(*cfg.percentile)
//	estimator = WithMargin(*cfg.marginFraction, estimator)
//
//	latestTimeSeries, err := p.GetRealtimeProvider().GetLatestTimeSeries(metricName, conditions)
//	if err != nil {
//		return nil, err
//	}
//	klog.Infof("percentilePredict len %v, latestTimeSeries: %+v", len(latestTimeSeries), latestTimeSeries)
//
//	estimationTimeSeries := make([]*common.TimeSeries, 0)
//	now := time.Now().Unix()
//	for _, ts := range latestTimeSeries {
//		key := prediction.AggregateSignalKey(metricName, ts.Labels)
//		s, exists := p.a.Load(key)
//		if !exists {
//			klog.Warningf("aggregate signal [%s] not found", key)
//			continue
//		}
//		sample := common.Sample{
//			Value:     estimator.GetEstimation(s.histogram),
//			Timestamp: now,
//		}
//		klog.Infof("key: %v, value: %v, time: %v", key, sample.Value, sample.Timestamp)
//		estimationTimeSeries = append(estimationTimeSeries, &common.TimeSeries{
//			Labels:  ts.Labels,
//			Samples: []common.Sample{sample},
//		})
//	}
//
//	return estimationTimeSeries, nil
//}

func (p *percentilePrediction) addSamples(queryExpr string) {
	labelsToTimeSeriesMap := map[string]*common.TimeSeries{}

	latestTimeSeries, err := p.GetRealtimeProvider().QueryLatestTimeSeries(queryExpr)
	if err != nil {
		logger.Error(err, "Failed to get latest time series")
		return
	}

	for i, ts := range latestTimeSeries {
		key := prediction.AggregateSignalKey("", ts.Labels)
		if len(ts.Samples) < 1 {
			logger.V(4).Info("Sample not found.", "key", key)
			continue
		}
		if _, exists := labelsToTimeSeriesMap[key]; exists {
			if labelsToTimeSeriesMap[key].Samples[0].Timestamp < ts.Samples[0].Timestamp {
				labelsToTimeSeriesMap[key] = latestTimeSeries[i]
			}
		} else {
			labelsToTimeSeriesMap[key] = latestTimeSeries[i]
		}
	}

	for _, ts := range labelsToTimeSeriesMap {
		sample := ts.Samples[0]

		logger.V(5).Info("Got latest time series sample", "queryExpr", queryExpr)

		key := prediction.AggregateSignalKey(queryExpr, ts.Labels)
		if _, exists := p.a.Load(key); !exists {
			p.a.Store(key, newAggregateSignal(queryExpr))
		}
		sampleTime := time.Unix(sample.Timestamp, 0)
		a, _ := p.a.Load(key)
		a.addSample(sampleTime, sample.Value)
	}
}
