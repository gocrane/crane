package percentile

import (
	"math"
	"sync"
	"time"

	"github.com/gocrane/crane/pkg/prediction"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/recommender/util"
)

type aggregateSignal struct {
	histogram         vpa.Histogram
	firstSampleTime   time.Time
	lastSampleTime    time.Time
	minSampleWeight   float64
	totalSamplesCount int
	creationTime      time.Time
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

type aggregateSignals struct {
	mu sync.Mutex
	callerMap map[string]/*expr*/map[string/*caller*/]struct{}
	signalMap map[string]/*expr*/map[string/*key*/]*aggregateSignal
}

func newAggregateSignals() aggregateSignals {
	return aggregateSignals{
		mu: sync.Mutex{},
		callerMap: map[string]map[string]struct{}{},
		signalMap: map[string]map[string]*aggregateSignal{},
	}
}

func (a *aggregateSignals) Add(qc prediction.QueryExprWithCaller) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.callerMap[qc.QueryExpr]; !exists {
		a.callerMap[qc.QueryExpr] = map[string]struct{}{}
	}

	if _, exists := a.callerMap[qc.QueryExpr][qc.Caller]; exists {
		return false
	}
	a.callerMap[qc.QueryExpr][qc.Caller] = struct{}{}

	if _, exists := a.signalMap[qc.QueryExpr]; !exists {
		a.signalMap[qc.QueryExpr] = map[string]*aggregateSignal{}
	}

	return true
}

func (a *aggregateSignals) Delete(qc prediction.QueryExprWithCaller) bool/*need clean or not*/ {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.callerMap[qc.QueryExpr]; !exists {
		return true
	}

	delete(a.callerMap[qc.QueryExpr], qc.Caller)
	if len(a.callerMap[qc.QueryExpr]) > 0 {
		return false
	}

	delete(a.callerMap, qc.QueryExpr)
	delete(a.signalMap, qc.QueryExpr)
	return true
}

func (a *aggregateSignals) SetSignal(queryExpr string, key string, signal *aggregateSignal) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.signalMap[queryExpr]; !exists {
		return
	}

	a.signalMap[queryExpr][key] = signal
}

func (a *aggregateSignals) GetSignal(queryExpr string, key string) *aggregateSignal {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.signalMap[queryExpr]; !exists {
		return nil
	}

	return a.signalMap[queryExpr][key]
}

func (a *aggregateSignals) GetOrStoreSignal(queryExpr string, key string, signal *aggregateSignal) *aggregateSignal {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.signalMap[queryExpr]; !exists {
		return nil
	}

	if _, exists := a.signalMap[queryExpr][key]; exists {
		return a.signalMap[queryExpr][key]
	}

	a.signalMap[queryExpr][key] = signal
	return signal
}
