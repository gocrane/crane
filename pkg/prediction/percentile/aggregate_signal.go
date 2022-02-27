package percentile

import (
	"math"
	"time"

	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/recommender/util"

	"github.com/gocrane/crane/pkg/common"
)

type aggregateSignal struct {
	histogram         vpa.Histogram
	firstSampleTime   time.Time
	lastSampleTime    time.Time
	minSampleWeight   float64
	totalSamplesCount int
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

func newAggregateSignal(c *internalConfig) *aggregateSignal {
	return &aggregateSignal{
		histogram:       vpa.NewHistogram(c.histogramOptions),
		minSampleWeight: c.minSampleWeight,
		creationTime:    time.Now(),
	}
}
