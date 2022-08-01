package dsp

import (
	"fmt"
	"sync"
	"time"

	"github.com/montanaflynn/stats"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
)

func fillMissingData(ts *common.TimeSeries, config *internalConfig, unit time.Duration) error {
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
			return fillMissingData(ts, config, unit)
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

func deTrend() error {
	return nil
}

func removeExtremeOutliers(ts *common.TimeSeries) error {
	values := make([]float64, len(ts.Samples))
	for i := 0; i < len(ts.Samples); i++ {
		values[i] = ts.Samples[i].Value
	}

	var highThreshold, lowThreshold float64
	var err error
	highThreshold, err = stats.Percentile(values, 99.9)
	if err != nil {
		return err
	}
	lowThreshold, err = stats.Percentile(values, 0.1)
	if err != nil {
		return err
	}

	for i := 1; i < len(ts.Samples); i++ {
		if ts.Samples[i].Value > highThreshold || ts.Samples[i].Value < lowThreshold {
			ts.Samples[i].Value = ts.Samples[i-1].Value
		}
	}
	return nil
}

func preProcessTimeSeries(ts *common.TimeSeries, config *internalConfig, unit time.Duration) error {
	var err error

	err = fillMissingData(ts, config, unit)
	if err != nil {
		return err
	}

	_ = deTrend()

	_ = removeExtremeOutliers(ts)

	return nil
}

func preProcessTimeSeriesList(tsList []*common.TimeSeries, config *internalConfig) ([]*common.TimeSeries, error) {
	var wg sync.WaitGroup

	n := len(tsList)
	wg.Add(n)
	tsCh := make(chan *common.TimeSeries, n)
	for i := range tsList {
		go func(ts *common.TimeSeries) {
			defer wg.Done()
			if err := preProcessTimeSeries(ts, config, Hour); err != nil {
				klog.ErrorS(err, "Dsp failed to pre process time series.")
			} else {
				tsCh <- ts
			}
		}(tsList[i])
	}
	wg.Wait()
	close(tsCh)

	tsList = make([]*common.TimeSeries, 0, n)
	for ts := range tsCh {
		tsList = append(tsList, ts)
	}

	return tsList, nil
}
