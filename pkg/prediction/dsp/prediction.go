package dsp

import (
	"fmt"
	"math"
	"sync"
	"time"

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
	a           aggregateSignalMap
	qr          config.Receiver
	modelConfig config.AlgorithmModelConfig
}

func NewPrediction(mc config.AlgorithmModelConfig) (prediction.Interface, error) {
	qb := config.NewBroadcaster()
	return &periodicSignalPrediction{
		GenericPrediction: prediction.NewGenericPrediction(qb),
		a:                 aggregateSignalMap{},
		qr:                qb.Listen(),
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
				logger.Error(err, "Dsp failed to pre process time series.")
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
		logger.Error(fmt.Errorf("history provider not provisioned"), "Run")
		return
	}

	for {
		// Waiting for a WithQuery request
		queryExpr := p.qr.Read().(string)
		logger.Info("Dsp received a WithQuery reques", "queryExpr", queryExpr)

		go func(queryExpr string) {
			ticker := time.NewTicker(p.modelConfig.UpdateInterval)
			defer ticker.Stop()
			for {
				logger.V(3).Info("Dsp start updateAggregateSignalsWithQuery", "query", queryExpr)
				if err := p.updateAggregateSignalsWithQuery(queryExpr); err != nil {
					logger.V(6).Info(fmt.Sprintf("Warning: updateAggregateSignalsWithQuery failed, err: %s", err.Error()))
				}
				select {
				case <-stopCh:
					return
				case <-ticker.C:
					continue
				}
			}
		}(queryExpr)
	}
}

func (p *periodicSignalPrediction) updateAggregateSignalsWithQuery(queryExpr string) error {
	// Query history data for prediction
	tsList, err := p.queryHistoryTimeSeries(queryExpr)
	if err != nil {
		logger.Error(err, "Dsp failed to get time series.", "queryExpr", queryExpr)
		return err
	}

	cfg := getInternalConfig(queryExpr)

	p.updateAggregateSignals(queryExpr, tsList, cfg)

	return nil
}

func (p *periodicSignalPrediction) queryHistoryTimeSeries(queryExpr string) ([]*common.TimeSeries, error) {
	if p.GetHistoryProvider() == nil {
		return nil, fmt.Errorf("history provider not provisioned")
	}

	config := getInternalConfig(queryExpr)

	end := time.Now().Truncate(config.historyResolution)
	start := end.Add(-config.historyDuration - time.Hour)

	tsList, err := p.GetHistoryProvider().QueryTimeSeries(queryExpr, start, end, config.historyResolution)
	if err != nil {
		logger.Error(err, "Dsp failed to query history time series.")
		return nil, err
	}

	logger.V(7).Info("Dsp queryHistoryTimeSeries", "tsList", tsList, "config", *config)

	return preProcessTimeSeriesList(tsList, config)
}

func (p *periodicSignalPrediction) updateAggregateSignals(id string, tsList []*common.TimeSeries, config *internalConfig) {
	var predictedTimeSeriesList []*common.TimeSeries

	for _, ts := range tsList {
		logger.V(9).Info("Dsp get time series.", "queryExpr", id, "tsSamples", ts.Samples, "tsLabels", ts.Labels)
		var chosenEstimator Estimator
		var signal *Signal
		var nCycles int
		var cycleDuration time.Duration = 0
		if isPeriodicTimeSeries(ts, config.historyResolution, Hour) {
			cycleDuration = Hour
			logger.Info("dsp time series is periodic.", "labels", ts.Labels, "cycleDuration", cycleDuration)
		} else if isPeriodicTimeSeries(ts, config.historyResolution, Day) {
			cycleDuration = Day
			logger.Info("dsp time series is periodic.", "labels", ts.Labels, "cycleDuration", cycleDuration)
		} else if isPeriodicTimeSeries(ts, config.historyResolution, Week) {
			cycleDuration = Week
			logger.Info("dsp time series is periodic.", "labels", ts.Labels, "cycleDuration", cycleDuration)
		} else {
			logger.Info("dsp time series is not periodic", "labels", ts.Labels)
		}

		if cycleDuration > 0 {
			signal = SamplesToSignal(ts.Samples, config.historyResolution)
			signal, nCycles = signal.Truncate(cycleDuration)
			if nCycles >= 2 {
				chosenEstimator = bestEstimator(config.estimators, signal, nCycles, cycleDuration)
			}
		}

		if chosenEstimator != nil {
			estimatedSignal := chosenEstimator.GetEstimation(signal, cycleDuration)
			intervalSeconds := int64(config.historyResolution.Seconds())
			nextTimestamp := ts.Samples[len(ts.Samples)-1].Timestamp + intervalSeconds

			samples := make([]common.Sample, len(estimatedSignal.Samples))
			for i := range estimatedSignal.Samples {
				samples[i] = common.Sample{
					Value:     estimatedSignal.Samples[i],
					Timestamp: nextTimestamp,
				}
				nextTimestamp += intervalSeconds
			}

			predictedTimeSeriesList = append(predictedTimeSeriesList, &common.TimeSeries{
				Labels:  ts.Labels,
				Samples: samples,
			})
		}
	}

	for i := range predictedTimeSeriesList {
		key := prediction.AggregateSignalKey(id, predictedTimeSeriesList[i].Labels)
		logger.Info("Dsp store aggregate signal key", "key", key)
		if _, exists := p.a.Load(key); !exists {
			logger.Info("dsp aggregateSignalKey added.", "key", key)
			p.a.Store(key, newAggregateSignal())
		}
		a, _ := p.a.Load(key)
		a.setPredictedTimeSeries(predictedTimeSeriesList[i])
	}
}

func bestEstimator(estimators []Estimator, signal *Signal, totalCycles int, cycleDuration time.Duration) Estimator {
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
			logger.Info("Dsp testing estimators ...", "estimator", estimators[i].String(), "error", pe)
			if err == nil && pe < minPE {
				minPE = pe
				bestEstimator = estimators[i]
			}
		}
	}

	logger.Info("Dsp got the best estimator", "error", minPE, "estimator", bestEstimator.String())
	return bestEstimator
}

func (p *periodicSignalPrediction) QueryPredictedTimeSeries(rawQuery string, startTime time.Time, endTime time.Time) ([]*common.TimeSeries, error) {
	if p.GetRealtimeProvider() == nil {
		return nil, fmt.Errorf("realtime data provider not set")
	}

	tsList, err := p.GetRealtimeProvider().QueryLatestTimeSeries(rawQuery)
	if err != nil {
		logger.Error(err, "Dsp failed to query latest time series")
		return nil, err
	}

	return p.getPredictedTimeSeriesList(rawQuery, tsList, startTime, endTime), nil
}

func (p *periodicSignalPrediction) QueryRealtimePredictedValues(queryExpr string) ([]*common.TimeSeries, error) {
	if p.GetRealtimeProvider() == nil {
		return nil, fmt.Errorf("realtime data provider not set")
	}
	config := getInternalConfig(queryExpr)

	tsList, err := p.GetRealtimeProvider().QueryLatestTimeSeries(queryExpr)
	if err != nil {
		logger.Error(err, "Dsp failed to query latest time series")
		return nil, err
	}

	now := time.Now()
	start := now.Truncate(config.historyResolution)
	end := start.Add(defaultFuture)

	predictedTimeSeries := p.getPredictedTimeSeriesList(queryExpr, tsList, start, end)

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

func (p *periodicSignalPrediction) getPredictedTimeSeriesList(id string, tsList []*common.TimeSeries, start, end time.Time) []*common.TimeSeries {
	var predictedTimeSeriesList []*common.TimeSeries

	for _, ts := range tsList {
		key := prediction.AggregateSignalKey(id, ts.Labels)
		logger.Info("Dsp get aggregate signal key", "key", key)
		a, exists := p.a.Load(key)
		if !exists {
			logger.Info("Dsp aggregate signal not found", "key", key)
			continue
		}

		var samples []common.Sample
		for _, sample := range a.predictedTimeSeries.Samples {
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
				Labels:  a.predictedTimeSeries.Labels,
				Samples: samples,
			})
		}

		logger.Info("Dsp got predicted samples.", "id", id, "len", len(samples), "labels", a.predictedTimeSeries.Labels)
	}
	return predictedTimeSeriesList
}
