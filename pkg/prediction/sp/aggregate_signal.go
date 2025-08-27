package sp

import (
	"k8s.io/klog/v2"
	"math"
	"time"

	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/recommender/util"

	"github.com/gocrane/crane/pkg/common"
)

// Sample information structure
type sampleInfo struct {
	time   time.Time
	value  float64
	weight float64
}

type aggregateSignal struct {
	histogram         vpa.Histogram
	firstSampleTime   time.Time
	lastSampleTime    time.Time
	minSampleWeight   float64
	totalSamplesCount int
	sampleInterval    time.Duration
	creationTime      time.Time
	labels            []common.Label
	historyLength     time.Duration
	// Sample queue (sorted by time, earliest at the front), default is added by time so no need to sort
	sampleQueue []sampleInfo
}

func (a *aggregateSignal) addSample(sampleTime time.Time, sampleValue float64) {
	// Calculate sample weight, emphasize the impact of large values, cover spike cases
	weight := math.Max(a.minSampleWeight, sampleValue)

	// Add new sample to histogram
	a.histogram.AddSample(sampleValue, weight, sampleTime)

	// Insert new sample into queue (sorted by time)
	newSample := sampleInfo{
		time:   sampleTime,
		value:  sampleValue,
		weight: weight,
	}

	a.sampleQueue = append(a.sampleQueue, newSample)

	// Update last sample time
	if a.lastSampleTime.Before(sampleTime) {
		a.lastSampleTime = sampleTime
	}

	// Update first sample time
	if a.firstSampleTime.IsZero() || a.firstSampleTime.After(sampleTime) {
		a.firstSampleTime = sampleTime
	}

	// Increase sample count
	a.totalSamplesCount++

	// Calculate expiration time point (current time - history length)
	expirationTime := time.Now().Add(-a.historyLength)

	// Remove expired samples (from the head of the queue, until encountering a non-expired sample)
	removedCount := 0
	for i := 0; i < len(a.sampleQueue); i++ {
		if a.sampleQueue[i].time.Before(expirationTime) {
			// Remove expired sample from histogram
			a.histogram.SubtractSample(
				a.sampleQueue[i].value,
				a.sampleQueue[i].weight,
				a.sampleQueue[i].time,
			)
			removedCount++
		} else {
			// Encounter the first non-expired sample, stop removing
			break
		}
	}

	// Update queue and count
	if removedCount > 0 {
		klog.V(4).Infof("Removed %v expired samples from the queue.", removedCount)
		a.sampleQueue = a.sampleQueue[removedCount:]
		a.totalSamplesCount -= removedCount
	}

	// Update first sample time
	if len(a.sampleQueue) > 0 {
		a.firstSampleTime = a.sampleQueue[0].time
	} else {
		a.firstSampleTime = time.Time{}
	}
}

// largest is 290 years, so it can not be overflow
func (a *aggregateSignal) GetAggregationWindowLength() time.Duration {
	return time.Duration(a.totalSamplesCount) * a.sampleInterval
}

func (a *aggregateSignal) IsSignalReady() bool {
	return a.GetAggregationWindowLength() >= a.historyLength/24
}

func newAggregateSignal(c *internalConfig) *aggregateSignal {
	var histogram vpa.Histogram
	if c.histogramDecayHalfLife == 0 {
		histogram = vpa.NewHistogram(c.histogramOptions)
	} else {
		histogram = vpa.NewDecayingHistogram(c.histogramOptions, c.histogramDecayHalfLife)
	}
	return &aggregateSignal{
		histogram:       histogram,
		minSampleWeight: c.minSampleWeight,
		creationTime:    time.Now(),
		sampleInterval:  c.sampleInterval,
		historyLength:   c.historyLength,
		sampleQueue:     []sampleInfo{}, // Initialize sample queue
	}
}
