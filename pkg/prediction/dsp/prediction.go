package dsp

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/prediction"
	"github.com/gocrane/crane/pkg/prediction/accuracy"
	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/providers"
)

var (
	Hour = time.Hour
	Day  = time.Hour * 24
	Week = Day * 7
)

const (
	defaultFuture = time.Hour
)

type periodicSignalPrediction struct {
	prediction.GenericPrediction
	a         aggregateSignals
	stopChMap sync.Map
	// record the query routine already started
	queryRoutines sync.Map
	modelConfig   config.AlgorithmModelConfig
}

func (p *periodicSignalPrediction) QueryPredictionStatus(ctx context.Context, metricNamer metricnaming.MetricNamer) (prediction.Status, error) {
	panic("implement me")
}

func NewPrediction(realtimeProvider providers.RealTime, historyProvider providers.History, mc config.AlgorithmModelConfig) prediction.Interface {
	withCh, delCh := make(chan prediction.QueryExprWithCaller), make(chan prediction.QueryExprWithCaller)
	return &periodicSignalPrediction{
		GenericPrediction: prediction.NewGenericPrediction(realtimeProvider, historyProvider, withCh, delCh),
		a:                 newAggregateSignals(),
		stopChMap:         sync.Map{},
		queryRoutines:     sync.Map{},
		modelConfig:       mc,
	}
}

func (p *periodicSignalPrediction) QueryRealtimePredictedValuesOnce(ctx context.Context, namer metricnaming.MetricNamer, config config.Config) ([]*common.TimeSeries, error) {
	panic("implement me")
}

// isPeriodicTimeSeries returns if time series has the specified periodicity
func isPeriodicTimeSeries(ts *common.TimeSeries, sampleInterval time.Duration, periodLength time.Duration) bool {
	signal := SamplesToSignal(ts.Samples, sampleInterval)
	return signal.IsPeriodic(periodLength)
}

func SamplesToSignal(samples []common.Sample, sampleInterval time.Duration) *Signal {
	values := make([]float64, len(samples))
	for i := range samples {
		values[i] = samples[i].Value
	}
	return &Signal{
		SampleRate: 1.0 / sampleInterval.Seconds(),
		Samples:    values,
	}
}

func (p *periodicSignalPrediction) Run(stopCh <-chan struct{}) {
	if p.GetHistoryProvider() == nil {
		klog.ErrorS(fmt.Errorf("history provider not provisioned"), "Failed to run periodicSignalPrediction.")
		return
	}

	go func() {
		for {
			// Waiting for a WithQuery request
			qc := <-p.WithCh
			// update if the query config updated, idempotent
			p.a.Add(qc)
			QueryExpr := qc.MetricNamer.BuildUniqueKey()

			if _, ok := p.queryRoutines.Load(QueryExpr); ok {
				continue
			}
			if _, ok := p.stopChMap.Load(QueryExpr); ok {
				continue
			}
			klog.V(6).InfoS("Register a query expression for prediction.", "queryExpr", QueryExpr, "caller", qc.Caller)

			go func(namer metricnaming.MetricNamer) {
				queryExpr := namer.BuildUniqueKey()
				p.queryRoutines.Store(queryExpr, struct{}{})
				ticker := time.NewTicker(p.modelConfig.UpdateInterval)
				defer ticker.Stop()

				v, _ := p.stopChMap.LoadOrStore(queryExpr, make(chan struct{}))
				predStopCh := v.(chan struct{})

				for {
					if err := p.updateAggregateSignalsWithQuery(namer); err != nil {
						klog.ErrorS(err, "Failed to updateAggregateSignalsWithQuery.")
					}

					select {
					case <-predStopCh:
						p.queryRoutines.Delete(queryExpr)
						klog.V(4).InfoS("Prediction routine stopped.", "queryExpr", queryExpr)
						return
					case <-ticker.C:
						continue
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

func (p *periodicSignalPrediction) updateAggregateSignalsWithQuery(namer metricnaming.MetricNamer) error {
	// Query history data for prediction
	maxAttempts := 10
	attempts := 0
	var tsList []*common.TimeSeries
	var err error
	queryExpr := namer.BuildUniqueKey()
	for attempts < maxAttempts {
		tsList, err = p.queryHistoryTimeSeries(namer)
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

	klog.V(6).InfoS("Update aggregate signals.", "queryExpr", queryExpr, "timeSeriesLength", len(tsList))

	cfg := p.a.GetConfig(queryExpr)

	p.updateAggregateSignals(queryExpr, tsList, cfg)

	return nil
}

func (p *periodicSignalPrediction) queryHistoryTimeSeries(namer metricnaming.MetricNamer) ([]*common.TimeSeries, error) {
	if p.GetHistoryProvider() == nil {
		return nil, fmt.Errorf("history provider not provisioned")
	}

	queryExpr := namer.BuildUniqueKey()
	config := p.a.GetConfig(queryExpr)

	end := time.Now().Truncate(config.historyResolution)
	start := end.Add(-config.historyDuration - time.Hour)

	tsList, err := p.GetHistoryProvider().QueryTimeSeries(namer, start, end, config.historyResolution)
	if err != nil {
		klog.ErrorS(err, "Failed to query history time series.")
		return nil, err
	}

	klog.V(6).InfoS("dsp queryHistoryTimeSeries", "timeSeriesList", tsList, "config", *config)

	return preProcessTimeSeriesList(tsList, config)
}

func (p *periodicSignalPrediction) updateAggregateSignals(queryExpr string, historyTimeSeriesList []*common.TimeSeries, config *internalConfig) {
	var predictedTimeSeriesList []*common.TimeSeries

	for _, ts := range historyTimeSeriesList {
		if klog.V(6).Enabled() {
			sampleData, err := json.Marshal(ts.Samples)
			klog.V(6).Infof("Got time series, queryExpr: %s, samples: %v, labels: %v, err: %v", queryExpr, string(sampleData), ts.Labels, err)
		}
		var chosenEstimator Estimator
		var signal *Signal
		var nPeriods int
		var periodLength time.Duration = 0
		if isPeriodicTimeSeries(ts, config.historyResolution, Day) {
			periodLength = Day
			klog.V(4).InfoS("This is a periodic time series.", "queryExpr", queryExpr, "labels", ts.Labels, "periodLength", periodLength)
		} else if isPeriodicTimeSeries(ts, config.historyResolution, Week) {
			periodLength = Week
			klog.V(4).InfoS("This is a periodic time series.", "queryExpr", queryExpr, "labels", ts.Labels, "periodLength", periodLength)
		} else {
			klog.V(4).InfoS("This is not a periodic time series.", "queryExpr", queryExpr, "labels", ts.Labels)
		}

		if periodLength > 0 {
			signal = SamplesToSignal(ts.Samples, config.historyResolution)
			signal, nPeriods = signal.Truncate(periodLength)
			if nPeriods >= 2 {
				chosenEstimator = bestEstimator(queryExpr, config.estimators, signal, nPeriods, periodLength)
			}
		}

		if chosenEstimator != nil {
			estimatedSignal := chosenEstimator.GetEstimation(signal, periodLength)
			intervalSeconds := int64(config.historyResolution.Seconds())
			nextTimestamp := ts.Samples[len(ts.Samples)-1].Timestamp + intervalSeconds

			n := len(estimatedSignal.Samples)
			samples := make([]common.Sample, n*nPeriods)
			for k := 0; k < nPeriods; k++ {
				for i := range estimatedSignal.Samples {
					samples[i+k*n] = common.Sample{
						Value:     estimatedSignal.Samples[i],
						Timestamp: nextTimestamp,
					}
					nextTimestamp += intervalSeconds
				}
			}

			predictedTimeSeriesList = append(predictedTimeSeriesList, &common.TimeSeries{
				Labels:  ts.Labels,
				Samples: samples,
			})
		}
	}

	signals := map[string]*aggregateSignal{}
	for i := range predictedTimeSeriesList {
		key := prediction.AggregateSignalKey(predictedTimeSeriesList[i].Labels)
		signal := newAggregateSignal()
		signal.setPredictedTimeSeries(predictedTimeSeriesList[i])
		signals[key] = signal
	}
	p.a.SetSignals(queryExpr, signals)
}

func bestEstimator(id string, estimators []Estimator, signal *Signal, nPeriods int, periodLength time.Duration) Estimator {
	samplesPerPeriod := len(signal.Samples) / nPeriods

	history := &Signal{
		SampleRate: signal.SampleRate,
		Samples:    signal.Samples[:(nPeriods-1)*samplesPerPeriod],
	}

	actual := &Signal{
		SampleRate: signal.SampleRate,
		Samples:    signal.Samples[(nPeriods-1)*samplesPerPeriod:],
	}

	minPE := math.MaxFloat64
	var bestEstimator Estimator
	for i := range estimators {
		estimated := estimators[i].GetEstimation(history, periodLength)
		if estimated != nil {
			pe, err := accuracy.PredictionError(actual.Samples, estimated.Samples)
			klog.V(6).InfoS("Testing estimators ...", "key", id, "estimator", estimators[i].String(), "pe", pe, "error", err)
			if err == nil && pe < minPE {
				minPE = pe
				bestEstimator = estimators[i]
			}
		}
	}

	klog.V(4).InfoS("Got the best estimator.", "key", id, "estimator", bestEstimator.String(), "minPE", minPE, "periods", nPeriods)
	return bestEstimator
}

func (p *periodicSignalPrediction) QueryPredictedTimeSeries(ctx context.Context, namer metricnaming.MetricNamer, startTime time.Time, endTime time.Time) ([]*common.TimeSeries, error) {
	return p.getPredictedTimeSeriesList(ctx, namer, startTime, endTime), nil
}

func (p *periodicSignalPrediction) QueryRealtimePredictedValues(ctx context.Context, namer metricnaming.MetricNamer) ([]*common.TimeSeries, error) {
	queryExpr := namer.BuildUniqueKey()
	config := p.a.GetConfig(queryExpr)

	now := time.Now()
	start := now.Truncate(config.historyResolution)
	end := start.Add(defaultFuture)

	predictedTimeSeries := p.getPredictedTimeSeriesList(ctx, namer, start, end)

	var realtimePredictedTimeSeries []*common.TimeSeries

	for _, ts := range predictedTimeSeries {
		if len(ts.Samples) < 1 {
			continue
		}
		maxValue := ts.Samples[0].Value
		for i := 1; i < len(ts.Samples); i++ {
			if maxValue < ts.Samples[i].Value {
				maxValue = ts.Samples[i].Value
			}
		}
		realtimePredictedTimeSeries = append(realtimePredictedTimeSeries, &common.TimeSeries{
			Labels:  ts.Labels,
			Samples: []common.Sample{{Value: maxValue, Timestamp: now.Unix()}},
		})
	}
	return realtimePredictedTimeSeries, nil
}

func (p *periodicSignalPrediction) getPredictedTimeSeriesList(ctx context.Context, namer metricnaming.MetricNamer, start, end time.Time) []*common.TimeSeries {
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
			for key, signal := range signals {
				var samples []common.Sample
				for _, sample := range signal.predictedTimeSeries.Samples {
					t := time.Unix(sample.Timestamp, 0)
					// Check if t is in [startTime, endTime]
					if !t.Before(start) && !t.After(end) {
						samples = append(samples, sample)
					} else if t.After(end) {
						break
					}
				}

				if len(samples) > 0 {
					predictedTimeSeriesList = append(predictedTimeSeriesList, &common.TimeSeries{
						Labels:  signal.predictedTimeSeries.Labels,
						Samples: samples,
					})
				}

				klog.InfoS("Got DSP predicted samples.", "queryExpr", queryExpr, "labels", key, "len", len(samples))
			}
			return predictedTimeSeriesList
		}
		select {
		case <-ctx.Done():
			klog.Infoln("Time out.")
			return predictedTimeSeriesList
		case <-ticker.C:
			continue
		}
	}
}

func (p *periodicSignalPrediction) Name() string {
	return "Periodic"
}
