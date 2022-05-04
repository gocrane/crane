package percentile

import (
	"math"
	"time"

	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/recommender/util"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/internal"
)

type aggregateSignal struct {
	histogram         vpa.Histogram
	firstSampleTime   time.Time
	lastSampleTime    time.Time
	minSampleWeight   float64
	totalSamplesCount uint64
	sampleInterval    time.Duration
	creationTime      time.Time
	labels            []common.Label
}

func (a *aggregateSignal) addSample(sampleTime time.Time, sampleValue float64) {
	a.histogram.AddSample(sampleValue, math.Max(a.minSampleWeight, sampleValue), sampleTime)
	if a.lastSampleTime.Before(sampleTime) {
		a.lastSampleTime = sampleTime
	}
	if a.firstSampleTime.IsZero() || a.firstSampleTime.After(sampleTime) {
		a.firstSampleTime = sampleTime
	}
	a.totalSamplesCount++
}

// largest is 290 years, so it can not be overflow
func (a *aggregateSignal) GetAggregationWindowLength() time.Duration {
	return time.Duration(a.totalSamplesCount) * a.sampleInterval
}

func (a *aggregateSignal) Expired(now time.Time) bool {
	if a.totalSamplesCount == 0 {
		return now.Sub(a.creationTime) >= a.GetAggregationWindowLength()
	}
	return now.Sub(a.lastSampleTime) >= a.GetAggregationWindowLength()
}

func newAggregateSignal(c *internalConfig) *aggregateSignal {
	return &aggregateSignal{
		histogram:       vpa.NewHistogram(c.histogramOptions),
		minSampleWeight: c.minSampleWeight,
		creationTime:    time.Now(),
		sampleInterval:  c.sampleInterval,
	}
}

func newAggregateSignalFromCheckpoint(c *internalConfig, checkpoint *internal.MetricNamerModelCheckpoint) (*aggregateSignal, error) {
	hist := vpa.NewHistogram(c.histogramOptions)
	err := hist.LoadFromCheckpoint(checkpoint.HistogramModel)
	if err != nil {
		return nil, err
	}
	return &aggregateSignal{
		histogram:         hist,
		firstSampleTime:   checkpoint.FirstSampleStart.Time,
		lastSampleTime:    checkpoint.LastSampleStart.Time,
		sampleInterval:    c.sampleInterval,
		minSampleWeight:   c.minSampleWeight,
		creationTime:      time.Now(),
		totalSamplesCount: checkpoint.TotalSamplesCount,
	}, nil
}
