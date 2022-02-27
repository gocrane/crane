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
	"github.com/gocrane/crane/pkg/prediction"
	"github.com/gocrane/crane/pkg/prediction/accuracy"
	"github.com/gocrane/crane/pkg/prediction/config"
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
	a           aggregateSignals
	stopChMap   sync.Map
	modelConfig config.AlgorithmModelConfig
}

func NewPrediction(mc config.AlgorithmModelConfig) (prediction.Interface, error) {
	withCh, delCh := make(chan prediction.QueryExprWithCaller), make(chan prediction.QueryExprWithCaller)
	return &periodicSignalPrediction{
		GenericPrediction: prediction.NewGenericPrediction(withCh, delCh),
		a:                 newAggregateSignals(),
		stopChMap:         sync.Map{},
		modelConfig:       mc,
	}, nil
}

func preProcessTimeSeriesList(tsList []*common.TimeSeries, config *internalConfig) ([]*common.TimeSeries, error) {
	var wg sync.WaitGroup

	n := len(tsList)
	wg.Add(n)
	tsCh := make(chan *common.TimeSeries, n)
	for _, ts := range tsList {
		go func(ts *common.TimeSeries) {
			defer wg.Done()
			if err := preProcessTimeSeries(ts, config, Hour); err != nil {
				klog.ErrorS(err, "Dsp failed to pre process time series.")
			} else {
				tsCh <- ts
			}
		}(ts)
	}
	wg.Wait()
	close(tsCh)

	tsList = make([]*common.TimeSeries, 0, n)
	for ts := range tsCh {
		tsList = append(tsList, ts)
	}
	wg.Wait()

	return tsList, nil
}

func preProcessTimeSeries(ts *common.TimeSeries, config *internalConfig, unit time.Duration) error {
	if ts == nil || len(ts.Samples) == 0 {
		return fmt.Errorf("empty time series")
	}

	intervalSeconds := int64(config.historyResolution.Seconds())

	for i := 1; i < len(ts.Samples); i++ {
		diff := ts.Samples[i].Timestamp - ts.Samples[i-1].Timestamp
		// If a gap in time series is larger than one hour,
		// drop all samples before [i].
		if diff > 3600 {
			ts.Samples = ts.Samples[i:]
			return preProcessTimeSeries(ts, config, unit)
		}

		// The samples should be in chronological order.
		// If the difference between two consecutive sample timestamps is not integral multiple of interval,
		// the time series is not valid.
		if diff%intervalSeconds != 0 || diff <= 0 {
			return fmt.Errorf("invalid time series")
		}
	}

	newSamples := []common.Sample{ts.Samples[0]}
	for i := 1; i < len(ts.Samples); i++ {
		times := (ts.Samples[i].Timestamp - ts.Samples[i-1].Timestamp) / intervalSeconds
		unitDiff := (ts.Samples[i].Value - ts.Samples[i-1].Value) / float64(times)
		// Fill the missing samples if any
		for j := int64(1); j < times; j++ {
			s := common.Sample{
				Value:     ts.Samples[i-1].Value + unitDiff*float64(j),
				Timestamp: ts.Samples[i-1].Timestamp + intervalSeconds*j,
			}
			newSamples = append(newSamples, s)
		}
		newSamples = append(newSamples, ts.Samples[i])
	}

	// Truncate samples of integral multiple of unit
	secondsPerUnit := int64(unit.Seconds())
	samplesPerUnit := int(secondsPerUnit / intervalSeconds)
	beginIndex := len(newSamples)
	for beginIndex-samplesPerUnit >= 0 {
		beginIndex -= samplesPerUnit
	}

	ts.Samples = newSamples[beginIndex:]

	return nil
}

// isPeriodicTimeSeries returns  time series with specified periodicity
func isPeriodicTimeSeries(ts *common.TimeSeries, sampleInterval time.Duration, cycleDuration time.Duration) bool {
	signal := SamplesToSignal(ts.Samples, sampleInterval)
	return signal.IsPeriodic(cycleDuration)
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
			if !p.a.Add(qc) {
				continue
			}

			if _, ok := p.stopChMap.Load(qc.QueryExpr); ok {
				continue
			}
			klog.V(6).InfoS("Register a query expression for prediction.", "queryExpr", qc.QueryExpr, "caller", qc.Caller)

			go func(queryExpr string) {
				ticker := time.NewTicker(p.modelConfig.UpdateInterval)
				defer ticker.Stop()

				v, _ := p.stopChMap.LoadOrStore(queryExpr, make(chan struct{}))
				predStopCh := v.(chan struct{})

				for {
					if err := p.updateAggregateSignalsWithQuery(queryExpr); err != nil {
						klog.ErrorS(err, "Failed to updateAggregateSignalsWithQuery.")
					}

					select {
					case <-predStopCh:
						klog.V(4).InfoS("Prediction routine stopped.", "queryExpr", queryExpr)
						return
					case <-ticker.C:
						continue
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

func (p *periodicSignalPrediction) updateAggregateSignalsWithQuery(queryExpr string) error {
	// Query history data for prediction
	maxAttempts := 10
	attempts := 0
	var tsList []*common.TimeSeries
	var err error
	for attempts < maxAttempts {
		tsList, err = p.queryHistoryTimeSeries(queryExpr)
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

func (p *periodicSignalPrediction) queryHistoryTimeSeries(queryExpr string) ([]*common.TimeSeries, error) {
	if p.GetHistoryProvider() == nil {
		return nil, fmt.Errorf("history provider not provisioned")
	}

	config := p.a.GetConfig(queryExpr)

	end := time.Now().Truncate(config.historyResolution)
	start := end.Add(-config.historyDuration - time.Hour)

	tsList, err := p.GetHistoryProvider().QueryTimeSeries(queryExpr, start, end, config.historyResolution)
	if err != nil {
		klog.ErrorS(err, "Failed to query history time series.")
		return nil, err
	}

	klog.V(6).InfoS("", "timeSeriesList", tsList, "config", *config)

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
		var nCycles int
		var cycleDuration time.Duration = 0
		if isPeriodicTimeSeries(ts, config.historyResolution, Hour) {
			cycleDuration = Hour
			klog.V(4).InfoS("This is a periodic time series.", "queryExpr", queryExpr, "labels", ts.Labels, "cycleDuration", cycleDuration)
		} else if isPeriodicTimeSeries(ts, config.historyResolution, Day) {
			cycleDuration = Day
			klog.V(4).InfoS("This is a periodic time series.", "queryExpr", queryExpr, "labels", ts.Labels, "cycleDuration", cycleDuration)
		} else if isPeriodicTimeSeries(ts, config.historyResolution, Week) {
			cycleDuration = Week
			klog.V(4).InfoS("This is a periodic time series.", "queryExpr", queryExpr, "labels", ts.Labels, "cycleDuration", cycleDuration)
		} else {
			klog.V(4).InfoS("This is not a periodic time series.", "queryExpr", queryExpr, "labels", ts.Labels)
		}

		if cycleDuration > 0 {
			signal = SamplesToSignal(ts.Samples, config.historyResolution)
			signal, nCycles = signal.Truncate(cycleDuration)
			if nCycles >= 2 {
				chosenEstimator = bestEstimator(queryExpr, config.estimators, signal, nCycles, cycleDuration)
			}
		}

		if chosenEstimator != nil {
			estimatedSignal := chosenEstimator.GetEstimation(signal, cycleDuration)
			intervalSeconds := int64(config.historyResolution.Seconds())
			nextTimestamp := ts.Samples[len(ts.Samples)-1].Timestamp + intervalSeconds

			// Hack(temporary):
			// Because the dsp predict only append one cycle points, when the cycle is hour, then estimate signal samples only contains at most one hour points
			// This leads to tsp predictWindowSeconds more than 3600 will be always out of date. because the predicted data end point timestamp is always ts.Samples[len(ts.Samples)-1].Timestamp+ Hour in one model update interval loop
			// If its cycle is hour, then we append 24 hour points to avoid the out of dated predicted data
			// Alternative option 1: Reduce predictWindowSeconds in tsp less than one hour and model update interval to one hour too.
			// Alternative option 2. Do not support hour cycle any more, because it is too short in production env. now the dsp can not handle hour cycle well.
			cycles := 1
			if cycleDuration == Hour {
				cycles = 24
			}
			n := len(estimatedSignal.Samples)
			samples := make([]common.Sample, n*cycles)
			for c := 0; c < cycles; c++ {
				for i := range estimatedSignal.Samples {
					samples[i+c*n] = common.Sample{
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

func bestEstimator(id string, estimators []Estimator, signal *Signal, totalCycles int, cycleDuration time.Duration) Estimator {
	samplesPerCycle := len(signal.Samples) / totalCycles

	history := &Signal{
		SampleRate: signal.SampleRate,
		Samples:    signal.Samples[:(totalCycles-1)*samplesPerCycle],
	}

	actual := &Signal{
		SampleRate: signal.SampleRate,
		Samples:    signal.Samples[(totalCycles-1)*samplesPerCycle:],
	}

	minPE := math.MaxFloat64
	var bestEstimator Estimator
	for i := range estimators {
		estimated := estimators[i].GetEstimation(history, cycleDuration)
		if estimated != nil {
			pe, err := accuracy.PredictionError(actual.Samples, estimated.Samples)
			klog.V(6).InfoS("Testing estimators ...", "key", id, "estimator", estimators[i].String(), "pe", pe, "error", err)
			if err == nil && pe < minPE {
				minPE = pe
				bestEstimator = estimators[i]
			}
		}
	}

	klog.V(4).InfoS("Got the best estimator.", "key", id, "estimator", bestEstimator.String(), "minPE", minPE, "totalCycles", totalCycles)
	return bestEstimator
}

func (p *periodicSignalPrediction) QueryPredictedTimeSeries(ctx context.Context, queryExpr string, startTime time.Time, endTime time.Time) ([]*common.TimeSeries, error) {
	return p.getPredictedTimeSeriesList(ctx, queryExpr, startTime, endTime), nil
}

func (p *periodicSignalPrediction) QueryRealtimePredictedValues(ctx context.Context, queryExpr string) ([]*common.TimeSeries, error) {
	config := p.a.GetConfig(
		queryExpr)

	now := time.Now()
	start := now.Truncate(config.historyResolution)
	end := start.Add(defaultFuture)

	predictedTimeSeries := p.getPredictedTimeSeriesList(ctx, queryExpr, start, end)

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

func (p *periodicSignalPrediction) getPredictedTimeSeriesList(ctx context.Context, queryExpr string, start, end time.Time) []*common.TimeSeries {
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

				klog.Info("Got DSP predicted samples.", "queryExpr", queryExpr, "labels", key, "len", len(samples))
			}
			return predictedTimeSeriesList
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

func (p *periodicSignalPrediction) Name() string {
	return "Periodic"
}
