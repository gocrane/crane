package percentile

import (
	"context"
	"math"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/prediction"
)

var _ prediction.Interface = &percentilePrediction{}

type percentilePrediction struct {
	prediction.GenericPrediction
	a         aggregateSignals
	stopChMap sync.Map
}

func (p *percentilePrediction) QueryPredictedTimeSeries(ctx context.Context, queryExpr string, startTime time.Time, endTime time.Time) ([]*common.TimeSeries, error) {
	var predictedTimeSeriesList []*common.TimeSeries
	cfg := p.a.GetConfig(queryExpr)
	tsList := p.getPredictedValues(ctx, queryExpr)
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

func (p *percentilePrediction) getPredictedValues(ctx context.Context, queryExpr string) []*common.TimeSeries {
	var predictedTimeSeriesList []*common.TimeSeries

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

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

func (p *percentilePrediction) QueryRealtimePredictedValues(ctx context.Context, queryExpr string) ([]*common.TimeSeries, error) {
	return p.getPredictedValues(ctx, queryExpr), nil
}

func NewPrediction() prediction.Interface {
	withCh, delCh := make(chan prediction.QueryExprWithCaller), make(chan prediction.QueryExprWithCaller)
	return &percentilePrediction{
		GenericPrediction: prediction.NewGenericPrediction(withCh, delCh),
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

			if _, ok := p.stopChMap.Load(qc.QueryExpr); ok {
				continue
			}

			klog.V(6).InfoS("Register a query expression for prediction.", "queryExpr", qc.QueryExpr, "caller", qc.Caller)

			if err := p.init(qc.QueryExpr); err != nil {
				klog.ErrorS(err, "Failed to init percentilePrediction.")
				continue
			}

			go func(queryExpr string) {
				if c := p.a.GetConfig(queryExpr); c != nil {
					ticker := time.NewTicker(c.sampleInterval)
					defer ticker.Stop()

					v, _ := p.stopChMap.LoadOrStore(queryExpr, make(chan struct{}))
					predStopCh := v.(chan struct{})

					for {
						p.addSamples(queryExpr)
						select {
						case <-predStopCh:
							klog.V(4).InfoS("Prediction routine stopped.", "queryExpr", queryExpr)
							return
						case <-ticker.C:
							continue
						}
					}
				}
			}(qc.QueryExpr)
		}
	}()

	go func() {
		for {
			qc := <-p.DelCh
			klog.V(4).InfoS("Unregister a query expression from prediction.", "queryExpr", qc.QueryExpr, "caller", qc.Caller)

			go func(qc prediction.QueryExprWithCaller) {
				if p.a.Delete(qc) {
					val, loaded := p.stopChMap.LoadAndDelete(qc.QueryExpr)
					if loaded {
						predStopCh := val.(chan struct{})
						predStopCh <- struct{}{}
					}
				}
			}(qc)
		}
	}()

	<-stopCh
}

func (p *percentilePrediction) queryHistoryTimeSeries(queryExpr string) ([]*common.TimeSeries, error) {
	if p.GetHistoryProvider() == nil {
		klog.Fatalln("History provider not found")
	}
	c := p.a.GetConfig(queryExpr)
	end := time.Now().Truncate(time.Minute)
	start := end.Add(-c.historyLength)
	historyTimeSeries, err := p.GetHistoryProvider().QueryTimeSeries(queryExpr, start, end, c.sampleInterval)
	if err != nil {
		klog.ErrorS(err, "Failed to query history time series.")
		return nil, err
	}
	return historyTimeSeries, nil
}

func (p *percentilePrediction) init(queryExpr string) error {
	// Query history data for prediction
	maxAttempts := 10
	attempts := 0
	var historyTimeSeriesList []*common.TimeSeries
	var err error
	for attempts < maxAttempts {
		historyTimeSeriesList, err = p.queryHistoryTimeSeries(queryExpr)
		if err != nil {
			attempts++
			t := time.Second * time.Duration(math.Pow(2., float64(attempts)))
			klog.ErrorS(err, "Failed to get time series.", "queryExpr", queryExpr, "attempts", attempts)
			time.Sleep(t)
		} else {
			break
		}
	}
	if attempts == maxAttempts {
		klog.Errorf("After attempting %d times, still cannot get history time series for query expression '%s'.", maxAttempts, queryExpr)
		return err
	}

	cfg := p.a.GetConfig(queryExpr)
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

func (p *percentilePrediction) addSamples(queryExpr string) {
	latestTimeSeriesList, err := p.GetRealtimeProvider().QueryLatestTimeSeries(queryExpr)
	if err != nil {
		klog.ErrorS(err, "Failed to get latest time series.")
		return
	}

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
