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
var keyAll = "__all__"

type percentilePrediction struct {
	prediction.GenericPrediction
	a aggregateSignals
	// record the query routine already started
	queryRoutines sync.Map
	stopChMap     sync.Map
}

func (p *percentilePrediction) QueryPredictionStatus(_ context.Context, metricNamer metricnaming.MetricNamer) (prediction.Status, error) {
	_, status := p.a.GetSignals(metricNamer.BuildUniqueKey())
	return status, nil
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

func (p *percentilePrediction) getPredictedValuesFromSignals(queryExpr string, signals map[string]*aggregateSignal, cfg *internalConfig) []*common.TimeSeries {
	var predictedTimeSeriesList []*common.TimeSeries

	if cfg == nil {
		cfg = p.a.GetConfig(queryExpr)
	}
	estimator := NewPercentileEstimator(cfg.percentile)
	estimator = WithMargin(cfg.marginFraction, estimator)
	estimator = WithTargetUtilization(cfg.targetUtilization, estimator)
	now := time.Now().Unix()

	if cfg.aggregated {
		signal := signals[keyAll]
		if signal != nil {
			sample := common.Sample{
				Value:     estimator.GetEstimation(signal.histogram),
				Timestamp: now,
			}
			predictedTimeSeriesList = append(predictedTimeSeriesList, &common.TimeSeries{
				Labels:  nil,
				Samples: []common.Sample{sample},
			})
		}
	} else {
		for key, signal := range signals {
			if key == keyAll {
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
	}

	return predictedTimeSeriesList
}

func (p *percentilePrediction) getPredictedValues(ctx context.Context, namer metricnaming.MetricNamer) []*common.TimeSeries {
	var predictedTimeSeriesList []*common.TimeSeries

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	queryExpr := namer.BuildUniqueKey()
	for {
		signals, status := p.a.GetSignals(queryExpr)
		if status == prediction.StatusUnknown {
			klog.V(4).InfoS("Aggregated has been deleted and unknown", "queryExpr", queryExpr)
			return predictedTimeSeriesList
		}
		if signals != nil && status == prediction.StatusReady {
			return p.getPredictedValuesFromSignals(queryExpr, signals, nil)
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
	queryExpr := namer.BuildUniqueKey()
	_, status := p.a.GetSignals(queryExpr)
	if status != prediction.StatusReady {
		return nil, fmt.Errorf("metric %v model status is %v, must be ready", queryExpr, status)
	}
	return p.getPredictedValues(ctx, namer), nil
}

// QueryRealtimePredictedValuesOnce is a one-off task, and the query will be deleted by the caller after the call. However, this query maybe has already been used by other callers,
// if so, we get the estimated value from the model directly.
// When the query is not found, we fetch the history data to build the histogram, and then get the estimated value by a stateless function as data processing way.
func (p *percentilePrediction) QueryRealtimePredictedValuesOnce(_ context.Context, namer metricnaming.MetricNamer, config config.Config) ([]*common.TimeSeries, error) {
	queryExpr := namer.BuildUniqueKey()

	cfg, err := makeInternalConfig(config.Percentile, config.InitMode)
	if err != nil {
		return nil, err
	}

	signals, status := p.a.GetSignals(queryExpr)
	if signals != nil && status == prediction.StatusReady {
		return p.getPredictedValuesFromSignals(queryExpr, signals, cfg), nil
	} else {
		// namer metric query is firstly registered by this caller
		// we first fetch history data to construct the histogram model, then get estimation.
		// it is just a stateless function, a data analyzing process, data in, then data out, no states.
		return p.process(namer, cfg)
	}
}

// process is a stateless function to get estimation of a metric series by constructing a histogram then get estimation data.
func (p *percentilePrediction) process(namer metricnaming.MetricNamer, cfg *internalConfig) ([]*common.TimeSeries, error) {
	var historyTimeSeriesList []*common.TimeSeries
	var err error
	queryExpr := namer.BuildUniqueKey()

	klog.V(4).Infof("process analyzing metric namer: %v, config: %+v", namer.BuildUniqueKey(), *cfg)

	historyTimeSeriesList, err = p.queryHistoryTimeSeries(namer, cfg)
	if err != nil {
		klog.Errorf("Failed to query history time series for query expression '%s'.", queryExpr)
		return nil, err
	}

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

	return p.getPredictedValuesFromSignals(queryExpr, signals, cfg), nil
}

func NewPrediction(realtimeProvider providers.RealTime, historyProvider providers.History) prediction.Interface {
	withCh, delCh := make(chan prediction.QueryExprWithCaller), make(chan prediction.QueryExprWithCaller)
	return &percentilePrediction{
		GenericPrediction: prediction.NewGenericPrediction(realtimeProvider, historyProvider, withCh, delCh),
		a:                 newAggregateSignals(),
		queryRoutines:     sync.Map{},
		stopChMap:         sync.Map{},
	}
}

func (p *percentilePrediction) Run(stopCh <-chan struct{}) {
	go func() {
		for {
			qc := <-p.WithCh
			// update if the query config updated, idempotent
			p.a.Add(qc)

			QueryExpr := qc.MetricNamer.BuildUniqueKey()

			if _, ok := p.queryRoutines.Load(QueryExpr); ok {
				klog.V(6).InfoS("Prediction percentile routine %v already registered.", "queryExpr", QueryExpr, "caller", qc.Caller)
				continue
			}

			if _, ok := p.stopChMap.Load(QueryExpr); ok {
				klog.V(6).InfoS("Prediction percentile routine %v already stopped.", "queryExpr", QueryExpr, "caller", qc.Caller)
				continue
			}

			klog.V(6).InfoS("Register a query expression for prediction.", "queryExpr", QueryExpr, "caller", qc.Caller)
			// todo: Do not block this management go routine here to do some time consuming operation.
			// We just init the signal and setting the status
			// we start the real time model updating directly. but there is a window time for each metricNamer in the algorithm config to ready status
			c := p.a.GetConfig(QueryExpr)

			var initError error
			switch c.initMode {
			case config.ModelInitModeLazyTraining:
				p.initByRealTimeProvider(qc.MetricNamer)
			case config.ModelInitModeCheckpoint:
				initError = p.initByCheckPoint(qc.MetricNamer)
			case config.ModelInitModeHistory:
				fallthrough
			default:
				// blocking
				initError = p.initFromHistory(qc.MetricNamer)
			}

			if initError != nil {
				klog.ErrorS(initError, "Failed to init percentilePrediction.")
				continue
			}

			// note: same query, but different config means different series analysis, GetConfig always return same config.
			// this is our default policy, one metric only has one config at a time.
			go func(namer metricnaming.MetricNamer) {
				queryExpr := namer.BuildUniqueKey()
				p.queryRoutines.Store(queryExpr, struct{}{})
				if c := p.a.GetConfig(queryExpr); c != nil {
					ticker := time.NewTicker(c.sampleInterval)
					defer ticker.Stop()

					v, _ := p.stopChMap.LoadOrStore(queryExpr, make(chan struct{}))
					predStopCh := v.(chan struct{})

					for {
						p.addSamples(namer)
						select {
						case <-predStopCh:
							p.queryRoutines.Delete(queryExpr)
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
func (p *percentilePrediction) initByRealTimeProvider(namer metricnaming.MetricNamer) {
	queryExpr := namer.BuildUniqueKey()
	cfg := p.a.GetConfig(queryExpr)
	if cfg.aggregated {
		signal := newAggregateSignal(cfg)

		p.a.SetSignalWithStatus(queryExpr, keyAll, signal, prediction.StatusInitializing)
	} else {
		signals := map[string]*aggregateSignal{}
		p.a.SetSignalsWithStatus(queryExpr, signals, prediction.StatusInitializing)
	}
}

// todo:
// nolint:unused
func (p *percentilePrediction) initByCheckPoint(_ metricnaming.MetricNamer) error {
	return fmt.Errorf("checkpoint not supported")
}

func (p *percentilePrediction) initFromHistory(namer metricnaming.MetricNamer) error {
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
		p.a.SetSignal(queryExpr, keyAll, signal)
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

	if c.aggregated {
		signal := p.a.GetSignal(queryExpr, keyAll)
		if signal == nil {
			return
		}
		// maybe we can use other aggregated way to deal with the container instances of the same container in different pods of the same workload,
		// aggregated by reducing all the samples to just one p99 or avg value and so on.
		// saw that when the workload is daemonset, container in different node has very different resource usage. this is unexpected in production, maybe the daemonset in different nodes has different loads.
		// NOTE: now it there are N instance of the workload, there are N samples in latest, then the aggregationWindowLength is N times growth to accumulate data fastly.
		// it is not a time dimension, but we use N samples of different container instances of the workload to represent the N intervals samples
		for _, ts := range latestTimeSeriesList {
			if len(ts.Samples) < 1 {
				klog.V(4).InfoS("Sample not found.", "key", keyAll)
				continue
			}
			sample := ts.Samples[len(ts.Samples)-1]
			sampleTime := time.Unix(sample.Timestamp, 0)
			signal.addSample(sampleTime, sample.Value)

			// current time is reach the window length of percentile need to accumulating data, the model is ready to do predict
			// all type need the aggregation window length is accumulating enough data.
			// History: if the container is newly, then there maybe has not enough history data, so we need accumulating to make the confidence more reliable
			// LazyTraining: directly accumulating data from real time metric provider until the data is enough
			// Checkpoint: directly recover the model from a checkpoint, and then updating the model until accumulated data is enough
			if signal.GetAggregationWindowLength() >= c.historyLength {
				p.a.SetSignalStatus(queryExpr, keyAll, prediction.StatusReady)
			}

			klog.V(6).InfoS("Sample added.", "sampleValue", sample.Value, "sampleTime", sampleTime, "queryExpr", queryExpr, "history", c.historyLength, "aggregationWindowLength", signal.GetAggregationWindowLength())
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

			// current time is reach the window length of percentile need to accumulating data, the model is ready to do predict
			// all type need the aggregation window length is accumulating enough data.
			// History: if the container is newly, then there maybe has not enough history data, so we need accumulating to make the confidence more reliable
			// LazyTraining: directly accumulating data from real time metric provider until the data is enough
			// Checkpoint: directly recover the model from a checkpoint, and then updating the model until accumulated data is enough
			if signal.GetAggregationWindowLength() >= c.historyLength {
				p.a.SetSignalStatus(queryExpr, key, prediction.StatusReady)
			}

			klog.V(6).InfoS("Sample added.", "sampleValue", sample.Value, "sampleTime", sampleTime, "queryExpr", queryExpr, "key", key, "history", c.historyLength, "aggregationWindowLength", signal.GetAggregationWindowLength())
		}
	}
}

func (p *percentilePrediction) Name() string {
	return "Percentile"
}
