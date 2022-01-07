package percentile

import (
	"fmt"
	"time"

	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/prediction"
	"github.com/gocrane/crane/pkg/prediction/config"
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

func (p *percentilePrediction) QueryPredictedTimeSeries(queryExpr string, startTime time.Time, endTime time.Time) ([]*common.TimeSeries, error) {
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
	klog.InfoS("Query latest time series.", "queryExpr", queryExpr, "latestTimeSeries", latestTimeSeries, "cfg", *cfg)

	estimatedTimeSeries := make([]*common.TimeSeries, 0)

	for _, ts := range latestTimeSeries {
		key := prediction.AggregateSignalKey(queryExpr, ts.Labels)
		s, exists := p.a.Load(key)
		if !exists {
			klog.Warningf("aggregate signal [%s] not found", key)
			continue
		}

		samples := GenSamplesFromWindow(estimator.GetEstimation(s.histogram), startTime, endTime, cfg.sampleInterval)
		estimatedTimeSeries = append(estimatedTimeSeries, &common.TimeSeries{
			Labels:  ts.Labels,
			Samples: samples,
		})
	}

	return estimatedTimeSeries, nil
}

func GenSamplesFromWindow(value float64, start time.Time, end time.Time, step time.Duration) []common.Sample {
	start = start.Truncate(step)
	var result []common.Sample
	for ts := start; ts.Before(end); ts = ts.Add(step) {
		result = append(result, common.Sample{Timestamp: ts.Unix(), Value: value})
	}
	return result
}

func (p *percentilePrediction) QueryRealtimePredictedValues(queryExpr string) ([]*common.TimeSeries, error) {
	if p.GetRealtimeProvider() == nil {
		return nil, fmt.Errorf("realtime data provider not set")
	}

	cfg := getInternalConfig(queryExpr)

	estimator := NewPercentileEstimator(cfg.percentile)
	estimator = WithMargin(cfg.marginFraction, estimator)

	now := time.Now().Unix()

	estimatedTimeSeries := make([]*common.TimeSeries, 0)

	if cfg.aggregated {
		key := prediction.AggregateSignalKey(queryExpr, nil)
		s, exists := p.a.Load(key)
		if !exists {
			klog.V(4).InfoS("Percentile aggregate signal not found", "key", key, "aggregated", true)
			return nil, nil
		}

		sample := common.Sample{
			Value:     estimator.GetEstimation(s.histogram),
			Timestamp: now,
		}

		estimatedTimeSeries = append(estimatedTimeSeries, &common.TimeSeries{
			Labels:  nil,
			Samples: []common.Sample{sample},
		})
	} else {
		latestTimeSeries, err := p.GetRealtimeProvider().QueryLatestTimeSeries(queryExpr)
		if err != nil {
			klog.ErrorS(err, "Failed to query latest time series.")
			return nil, err
		}
		klog.V(6).InfoS("Query latest time series.", "latestTimeSeries", latestTimeSeries)

		for _, ts := range latestTimeSeries {
			key := prediction.AggregateSignalKey(queryExpr, ts.Labels)
			s, exists := p.a.Load(key)
			if !exists {
				klog.V(4).InfoS("Aggregate signal not found.", "key", key, "aggregated", false)
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
	}

	return estimatedTimeSeries, nil
}

func NewPrediction() prediction.Interface {
	//mb := config.NewBroadcaster()
	qb := config.NewBroadcaster()
	return &percentilePrediction{
		GenericPrediction: prediction.NewGenericPrediction(qb),
		a:                 aggregateSignalMap{},
		qr:                qb.Listen(),
	}
}

func (p *percentilePrediction) Run(stopCh <-chan struct{}) {
	for {
		// Waiting for a new config
		expr := p.qr.Read().(string)

		if err := p.init(expr); err != nil {
			klog.ErrorS(err, "Failed to init percentilePrediction.")
			continue
		}

		go func(expr string) {
			if expr == "" {
				return
			}
			if c := getInternalConfig(expr); c != nil {
				ticker := time.NewTicker(c.sampleInterval)
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
		}(expr)
	}
}

func (p *percentilePrediction) init(queryExpr string) error {
	if p.GetHistoryProvider() == nil {
		return fmt.Errorf("history provider not found")
	}
	c := getInternalConfig(queryExpr)

	end := time.Now().Truncate(time.Minute)
	start := end.Add(-c.historyLength)

	historyTimeSeries, err := p.GetHistoryProvider().QueryTimeSeries(queryExpr, start, end, c.sampleInterval)
	if err != nil {
		klog.ErrorS(err, "Failed to query history time series.")
		return err
	}

	if c.aggregated {
		key := prediction.AggregateSignalKey(queryExpr, nil)
		if _, exists := p.a.Load(key); exists {
			p.a.Delete(key)
		}
		p.a.Store(key, newAggregateSignal(c))
		a, _ := p.a.Load(key)
		for _, ts := range historyTimeSeries {
			for _, s := range ts.Samples {
				t := time.Unix(s.Timestamp, 0)
				a.addSample(t, s.Value)
			}
		}
	} else {
		labelsToTimeSeriesMap := map[string]*common.TimeSeries{}
		for i, ts := range historyTimeSeries {
			key := prediction.AggregateSignalKey(queryExpr, ts.Labels)
			if len(ts.Samples) < 1 {
				continue
			}
			labelsToTimeSeriesMap[key] = historyTimeSeries[i]
		}

		for key, ts := range labelsToTimeSeriesMap {
			if _, exists := p.a.Load(key); exists {
				p.a.Delete(key)
			}
			p.a.Store(key, newAggregateSignal(c))
			a, _ := p.a.Load(key)
			for _, s := range ts.Samples {
				t := time.Unix(s.Timestamp, 0)
				a.addSample(t, s.Value)
			}
		}
	}

	return nil
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
	latestTimeSeries, err := p.GetRealtimeProvider().QueryLatestTimeSeries(queryExpr)
	if err != nil {
		klog.ErrorS(err, "Failed to get latest time series.")
		return
	}

	c := getInternalConfig(queryExpr)

	if c.aggregated {
		key := prediction.AggregateSignalKey(queryExpr, nil)
		for _, ts := range latestTimeSeries {
			if len(ts.Samples) < 1 {
				klog.V(4).InfoS("Sample not found.", "key", key)
				continue
			}

			if _, exists := p.a.Load(key); !exists {
				p.a.Store(key, newAggregateSignal(c))
			}

			sample := ts.Samples[len(ts.Samples)-1]
			sampleTime := time.Unix(sample.Timestamp, 0)
			a, _ := p.a.Load(key)
			a.addSample(sampleTime, sample.Value)
			klog.V(6).InfoS("Sample added.", "sampleValue", sample.Value, "sampleTime", sampleTime, "queryExpr", queryExpr)
		}
	} else {
		labelsToTimeSeriesMap := map[string]*common.TimeSeries{}
		for i, ts := range latestTimeSeries {
			key := prediction.AggregateSignalKey(queryExpr, ts.Labels)
			if len(ts.Samples) < 1 {
				klog.V(4).InfoS("Sample not found.", "key", key)
				continue
			}
			labelsToTimeSeriesMap[key] = latestTimeSeries[i]
		}
		for key, ts := range labelsToTimeSeriesMap {
			sample := ts.Samples[len(ts.Samples)-1]
			klog.V(6).Info("Got latest time series sample.", "key", key, "sample", sample)
			if _, exists := p.a.Load(key); !exists {
				p.a.Store(key, newAggregateSignal(c))
			}
			sampleTime := time.Unix(sample.Timestamp, 0)
			a, _ := p.a.Load(key)
			a.addSample(sampleTime, sample.Value)
		}
	}
}

func (p *percentilePrediction) Name() string {
	return "Percentile"
}
