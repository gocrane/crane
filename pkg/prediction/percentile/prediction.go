package percentile

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/prediction"
	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/providers"
)

var _ prediction.Interface = &percentilePrediction{}

type percentilePrediction struct {
	prediction.GenericPrediction
	a         aggregateSignals
	stopChMap sync.Map
}

func (p *percentilePrediction) QueryPredictedTimeSeries(ctx context.Context, namer metricnaming.MetricNamer, startTime time.Time, endTime time.Time) ([]*common.TimeSeries, error) {
	var predictedTimeSeriesList []*common.TimeSeries
	queryExpr := namer.BuildUniqueKey()
	cfg := p.a.GetConfig(queryExpr)
	tsList := p.getPredictedValues(ctx, namer)
	for _, ts := range tsList {
		n := len(ts.Samples)
		if n > 0 {
			samples := generateSamplesFromWindow(ts.Samples[n-1].Value, startTime, endTime, cfg.sampleInterval)
			predictedTimeSeriesList = append(predictedTimeSeriesList, &common.TimeSeries{
				Labels:  ts.Labels,
				Samples: samples,
			})
		}
	}
	return predictedTimeSeriesList, nil
}

func generateSamplesFromWindow(value float64, start time.Time, end time.Time, step time.Duration) []common.Sample {
	start = start.Truncate(step)
	var result []common.Sample
	for ts := start; ts.Before(end); ts = ts.Add(step) {
		result = append(result, common.Sample{Timestamp: ts.Unix(), Value: value})
	}
	return result
}

func (p *percentilePrediction) getPredictedValues(ctx context.Context, namer metricnaming.MetricNamer) []*common.TimeSeries {
	var predictedTimeSeriesList []*common.TimeSeries

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	queryExpr := namer.BuildUniqueKey()
	for {
		signals, status := p.a.GetSignals(queryExpr)
		if status == prediction.StatusDeleted {
			klog.V(4).InfoS("Aggregated has been deleted.", "queryExpr", queryExpr)
			return predictedTimeSeriesList
		}
		if signals != nil && status == prediction.StatusReady {
			cfg := p.a.GetConfig(queryExpr)
			estimator := NewPercentileEstimator(cfg.percentile)
			estimator = WithMargin(cfg.marginFraction, estimator)
			now := time.Now().Unix()

			if cfg.aggregated {
				key := "__all__"
				signal := signals[key]
				if signal == nil {
					return nil
				}
				sample := common.Sample{
					Value:     estimator.GetEstimation(signal.histogram),
					Timestamp: now,
				}
				predictedTimeSeriesList = append(predictedTimeSeriesList, &common.TimeSeries{
					Labels:  nil,
					Samples: []common.Sample{sample},
				})
				return predictedTimeSeriesList
			} else {
				for key, signal := range signals {
					if key == "__all__" {
						continue
					}
					sample := common.Sample{
						Value:     estimator.GetEstimation(signal.histogram),
						Timestamp: now,
					}
					predictedTimeSeriesList = append(predictedTimeSeriesList, &common.TimeSeries{
						Labels:  signal.labels,
						Samples: []common.Sample{sample},
					})
				}
				return predictedTimeSeriesList
			}
		}
		select {
		case <-ctx.Done():
			klog.Info("Time out.")
			return predictedTimeSeriesList
		case <-ticker.C:
			continue
		}
	}
}

func (p *percentilePrediction) QueryRealtimePredictedValues(ctx context.Context, namer metricnaming.MetricNamer) ([]*common.TimeSeries, error) {
	return p.getPredictedValues(ctx, namer), nil
}

// once task, it is only called once then caller will delete the query after call, but this query maybe used by other callers,
// so when there has already registered this namer query, we get the estimated value from the model directly.
// when there is no this namer query in state, we fetch history data to recover the histogram model then get the estimated value by a stateless function as data processing way.
func (p *percentilePrediction) QueryRealtimePredictedValuesOnce(ctx context.Context, namer metricnaming.MetricNamer, config config.Config) ([]*common.TimeSeries, error) {
	var predictedTimeSeriesList []*common.TimeSeries

	queryExpr := namer.BuildUniqueKey()

	signals, status := p.a.GetSignals(queryExpr)
	if signals != nil && status == prediction.StatusReady {
		cfg := p.a.GetConfig(queryExpr)
		estimator := NewPercentileEstimator(cfg.percentile)
		estimator = WithMargin(cfg.marginFraction, estimator)
		now := time.Now().Unix()

		if cfg.aggregated {
			key := "__all__"
			signal := signals[key]
			if signal == nil {
				return nil, fmt.Errorf("No signal key %v found", key)
			}
			sample := common.Sample{
				Value:     estimator.GetEstimation(signal.histogram),
				Timestamp: now,
			}
			predictedTimeSeriesList = append(predictedTimeSeriesList, &common.TimeSeries{
				Labels:  nil,
				Samples: []common.Sample{sample},
			})
			return predictedTimeSeriesList, nil
		} else {
			for key, signal := range signals {
				if key == "__all__" {
					continue
				}
				sample := common.Sample{
					Value:     estimator.GetEstimation(signal.histogram),
					Timestamp: now,
				}
				predictedTimeSeriesList = append(predictedTimeSeriesList, &common.TimeSeries{
					Labels:  signal.labels,
					Samples: []common.Sample{sample},
				})
			}
			return predictedTimeSeriesList, nil
		}
	} else {
		// namer metric query is firstly registered by this caller
		// we first fetch history data to construct the histogram model, then get estimation.
		// it is just a stateless function, a data analyzing process, data in, then data out, no states.
		return p.process(namer, config)
	}
}

// process is a stateless function to get estimation of a metric series by constructing a histogram then get estimation data.
func (p *percentilePrediction) process(namer metricnaming.MetricNamer, config config.Config) ([]*common.TimeSeries, error) {
	var predictedTimeSeriesList []*common.TimeSeries
	var historyTimeSeriesList []*common.TimeSeries
	var err error
	queryExpr := namer.BuildUniqueKey()
	cfg, err := makeInternalConfig(config.Percentile)
	if err != nil {
		return nil, err
	}
	klog.V(4).Infof("process analyzing metric namer: %v, config: %+v", namer.BuildUniqueKey(), *cfg)

	historyTimeSeriesList, err = p.queryHistoryTimeSeries(namer, cfg)
	if err != nil {
		klog.Errorf("Failed to query history time series for query expression '%s'.", queryExpr)
		return nil, err
	}

	signals := map[string]*aggregateSignal{}
	keyAll := "__all__"
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

	estimator := NewPercentileEstimator(cfg.percentile)
	estimator = WithMargin(cfg.marginFraction, estimator)
	now := time.Now().Unix()

	if cfg.aggregated {
		signal := signals[keyAll]
		if signal == nil {
			return nil, fmt.Errorf("No signal key %v found", keyAll)
		}
		sample := common.Sample{
			Value:     estimator.GetEstimation(signal.histogram),
			Timestamp: now,
		}
		predictedTimeSeriesList = append(predictedTimeSeriesList, &common.TimeSeries{
			Labels:  nil,
			Samples: []common.Sample{sample},
		})
		return predictedTimeSeriesList, nil
	} else {
		for key, signal := range signals {
			if key == "__all__" {
				continue
			}
			sample := common.Sample{
				Value:     estimator.GetEstimation(signal.histogram),
				Timestamp: now,
			}
			predictedTimeSeriesList = append(predictedTimeSeriesList, &common.TimeSeries{
				Labels:  signal.labels,
				Samples: []common.Sample{sample},
			})
		}
		return predictedTimeSeriesList, nil
	}
}

func NewPrediction(realtimeProvider providers.RealTime, historyProvider providers.History) prediction.Interface {
	withCh, delCh := make(chan prediction.QueryExprWithCaller), make(chan prediction.QueryExprWithCaller)
	return &percentilePrediction{
		GenericPrediction: prediction.NewGenericPrediction(realtimeProvider, historyProvider, withCh, delCh),
		a:                 newAggregateSignals(),
		stopChMap:         sync.Map{},
	}
}

func (p *percentilePrediction) Run(stopCh <-chan struct{}) {
	go func() {
		for {
			qc := <-p.WithCh
			if !p.a.Add(qc) {
				continue
			}

			QueryExpr := qc.MetricNamer.BuildUniqueKey()
			if _, ok := p.stopChMap.Load(QueryExpr); ok {
				continue
			}

			klog.V(6).InfoS("Register a query expression for prediction.", "queryExpr", QueryExpr, "caller", qc.Caller)
			// todo: Do not block this management go routine here to do some time consuming operation.
			// We just init the signal and setting the status
			// we start the real time model updating directly. but there is a window time for each metricNamer in the algorithm config to ready status
			if err := p.init(qc.MetricNamer); err != nil {
				klog.ErrorS(err, "Failed to init percentilePrediction.")
				continue
			}

			// bug here:
			// 1. first, judge the  metric is already started, if so, we do not start the updating model routine again.
			// 2. same query, but different config means different series analysis, but here GetConfig always return same config.
			go func(namer metricnaming.MetricNamer) {
				queryExpr := namer.BuildUniqueKey()
				if c := p.a.GetConfig(queryExpr); c != nil {
					ticker := time.NewTicker(c.sampleInterval)
					defer ticker.Stop()

					v, _ := p.stopChMap.LoadOrStore(queryExpr, make(chan struct{}))
					predStopCh := v.(chan struct{})

					for {
						p.addSamples(namer)
						select {
						case <-predStopCh:
							klog.V(4).InfoS("Prediction routine stopped.", "queryExpr", queryExpr)
							return
						case <-ticker.C:
							continue
						}
					}
				}
			}(qc.MetricNamer)
		}
	}()

	go func() {
		for {
			qc := <-p.DelCh
			QueryExpr := qc.MetricNamer.BuildUniqueKey()
			klog.V(4).InfoS("Unregister a query expression from prediction.", "queryExpr", QueryExpr, "caller", qc.Caller)

			go func(qc prediction.QueryExprWithCaller) {
				if p.a.Delete(qc) {
					val, loaded := p.stopChMap.LoadAndDelete(QueryExpr)
					if loaded {
						predStopCh := val.(chan struct{})
						predStopCh <- struct{}{}
					}
				}
			}(qc)
		}
	}()

	klog.Infof("predictor %v started", p.Name())

	<-stopCh

	klog.Infof("predictor %v stopped", p.Name())

}

func (p *percentilePrediction) queryHistoryTimeSeries(namer metricnaming.MetricNamer, c *internalConfig) ([]*common.TimeSeries, error) {
	if p.GetHistoryProvider() == nil {
		klog.Fatalln("History provider not found")
	}
	end := time.Now().Truncate(time.Minute)
	start := end.Add(-c.historyLength)
	historyTimeSeries, err := p.GetHistoryProvider().QueryTimeSeries(namer, start, end, c.sampleInterval)
	if err != nil {
		klog.ErrorS(err, "Failed to query history time series.")
		return nil, err
	}
	return historyTimeSeries, nil
}

// Lazy training the histogram model. we do not init from History Provider such as prometheus because prometheus's poor performance issue.
// Rather, we recover or init a complete available histogram model by fetching real time data point continuously until it is enough time to do estimation.
// So, we can set a waiting time for the model trained completed. Because percentile is only used for request resource & resource estimation.
// Because of the scenes, we do not need it give a result fastly after service start, we can tolerate it has some days delaying for collecting more data.
// nolint:unused
func (p *percentilePrediction) initByRealTimeProvider(namer metricnaming.MetricNamer) error {
	queryExpr := namer.BuildUniqueKey()
	cfg := p.a.GetConfig(queryExpr)
	latestTimeSeriesList, err := p.GetRealtimeProvider().QueryLatestTimeSeries(namer)
	if err != nil {
		klog.ErrorS(err, "Failed to get latest time series.")
		return err
	}
	if cfg.aggregated {
		signal := newAggregateSignal(cfg)
		for _, ts := range latestTimeSeriesList {
			for _, s := range ts.Samples {
				t := time.Unix(s.Timestamp, 0)
				signal.addSample(t, s.Value)
			}
		}
		p.a.SetSignal(queryExpr, "__all__", signal)
	} else {
		signals := map[string]*aggregateSignal{}
		for _, ts := range latestTimeSeriesList {
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
		p.a.SetSignals(queryExpr, signals)
	}

	return nil
}

// todo:
// nolint:unused
func (p *percentilePrediction) initByCheckPoint(namer metricnaming.MetricNamer) error {
	return nil
}

func (p *percentilePrediction) init(namer metricnaming.MetricNamer) error {
	queryExpr := namer.BuildUniqueKey()
	cfg := p.a.GetConfig(queryExpr)
	// Query history data for prediction
	var historyTimeSeriesList []*common.TimeSeries
	var err error
	historyTimeSeriesList, err = p.queryHistoryTimeSeries(namer, cfg)
	if err != nil {
		klog.Errorf("Failed to query history time series for query expression '%s'.", queryExpr)
		return err
	}

	if cfg.aggregated {
		signal := newAggregateSignal(cfg)
		for _, ts := range historyTimeSeriesList {
			for _, s := range ts.Samples {
				t := time.Unix(s.Timestamp, 0)
				signal.addSample(t, s.Value)
			}
		}
		p.a.SetSignal(queryExpr, "__all__", signal)
	} else {
		signals := map[string]*aggregateSignal{}
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
		p.a.SetSignals(queryExpr, signals)
	}

	return nil
}

func (p *percentilePrediction) addSamples(namer metricnaming.MetricNamer) {
	latestTimeSeriesList, err := p.GetRealtimeProvider().QueryLatestTimeSeries(namer)
	if err != nil {
		klog.ErrorS(err, "Failed to get latest time series.")
		return
	}

	queryExpr := namer.BuildUniqueKey()
	c := p.a.GetConfig(queryExpr)

	if _, status := p.a.GetSignals(queryExpr); status != prediction.StatusReady {
		klog.InfoS("Aggregate signals not ready.", "queryExpr", queryExpr, "status", status)
		return
	}

	if c.aggregated {
		key := "__all__"
		signal := p.a.GetSignal(queryExpr, key)
		if signal == nil {
			return
		}
		for _, ts := range latestTimeSeriesList {
			if len(ts.Samples) < 1 {
				klog.V(4).InfoS("Sample not found.", "key", key)
				continue
			}
			sample := ts.Samples[len(ts.Samples)-1]
			sampleTime := time.Unix(sample.Timestamp, 0)
			signal.addSample(sampleTime, sample.Value)
			klog.V(6).InfoS("Sample added.", "sampleValue", sample.Value, "sampleTime", sampleTime, "queryExpr", queryExpr)
		}
	} else {
		// todo: find a way to remove the labels key, although we do not really use it now.
		for _, ts := range latestTimeSeriesList {
			key := prediction.AggregateSignalKey(ts.Labels)
			if len(ts.Samples) < 1 {
				klog.V(4).InfoS("Sample not found.", "queryExpr", queryExpr, "key", key)
				continue
			}

			signal := p.a.GetOrStoreSignal(queryExpr, key, newAggregateSignal(c))
			if signal == nil {
				continue
			}
			sample := ts.Samples[len(ts.Samples)-1]
			sampleTime := time.Unix(sample.Timestamp, 0)
			signal.addSample(sampleTime, sample.Value)
			if len(signal.labels) == 0 {
				signal.labels = ts.Labels
			}
			klog.V(6).InfoS("Sample added.", "sampleValue", sample.Value, "sampleTime", sampleTime, "queryExpr", queryExpr, "key", key)
		}
	}
}

func (p *percentilePrediction) Name() string {
	return "Percentile"
}
