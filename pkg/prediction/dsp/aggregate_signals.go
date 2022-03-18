package dsp

import (
	"sync"

	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/prediction"
)

type aggregateSignals struct {
	mutex     sync.RWMutex
	callerMap map[string] /*expr*/ map[string] /*caller*/ struct{}
	signalMap map[string] /*expr*/ map[string] /*key*/ *aggregateSignal
	statusMap map[string] /*expr*/ prediction.Status
	configMap map[string]*internalConfig
}

func newAggregateSignals() aggregateSignals {
	return aggregateSignals{
		mutex:     sync.RWMutex{},
		callerMap: map[string]map[string]struct{}{},
		signalMap: map[string]map[string]*aggregateSignal{},
		statusMap: map[string]prediction.Status{},
		configMap: map[string]*internalConfig{},
	}
}

func (a *aggregateSignals) Add(qc prediction.QueryExprWithCaller) bool {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	QueryExpr := qc.MetricNamer.BuildUniqueKey()
	if qc.Config.DSP != nil {
		cfg, err := makeInternalConfig(qc.Config.DSP)
		if err != nil {
			klog.ErrorS(err, "Failed to make internal config.", "queryExpr", QueryExpr)
		} else {
			a.configMap[QueryExpr] = cfg
		}
	}

	if _, exists := a.callerMap[QueryExpr]; !exists {
		a.callerMap[QueryExpr] = map[string]struct{}{}
	}

	if status, exists := a.statusMap[QueryExpr]; !exists || status == prediction.StatusDeleted {
		a.statusMap[QueryExpr] = prediction.StatusNotStarted
	}

	if _, exists := a.callerMap[QueryExpr][qc.Caller]; exists {
		return false
	}
	a.callerMap[QueryExpr][qc.Caller] = struct{}{}

	if _, exists := a.signalMap[QueryExpr]; !exists {
		a.signalMap[QueryExpr] = map[string]*aggregateSignal{}
		return true
	}

	return false
}

func (a *aggregateSignals) Delete(qc prediction.QueryExprWithCaller) bool /*need clean or not*/ {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	QueryExpr := qc.MetricNamer.BuildUniqueKey()
	if _, exists := a.callerMap[QueryExpr]; !exists {
		return true
	}

	delete(a.callerMap[QueryExpr], qc.Caller)
	if len(a.callerMap[QueryExpr]) > 0 {
		return false
	}

	delete(a.callerMap, QueryExpr)
	delete(a.signalMap, QueryExpr)
	delete(a.configMap, QueryExpr)
	a.statusMap[QueryExpr] = prediction.StatusDeleted
	return true
}

func (a *aggregateSignals) GetConfig(queryExpr string) *internalConfig {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	if a.configMap[queryExpr] != nil {
		return a.configMap[queryExpr]
	}
	return &defaultInternalConfig
}

func (a *aggregateSignals) SetSignal(queryExpr string, key string, signal *aggregateSignal) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if _, exists := a.signalMap[queryExpr]; !exists {
		return
	}

	a.signalMap[queryExpr][key] = signal
}

func (a *aggregateSignals) GetSignal(queryExpr string, key string) *aggregateSignal {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	if _, exists := a.signalMap[queryExpr]; !exists {
		return nil
	}

	return a.signalMap[queryExpr][key]
}

func (a *aggregateSignals) GetOrStoreSignal(queryExpr string, key string, signal *aggregateSignal) *aggregateSignal {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if _, exists := a.signalMap[queryExpr]; !exists {
		return nil
	}

	if _, exists := a.signalMap[queryExpr][key]; exists {
		return a.signalMap[queryExpr][key]
	}

	a.signalMap[queryExpr][key] = signal
	return signal
}

func (a *aggregateSignals) SetSignals(queryExpr string, signals map[string]*aggregateSignal) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if _, exists := a.signalMap[queryExpr]; !exists {
		return
	}
	for k, v := range signals {
		a.signalMap[queryExpr][k] = v
	}
	a.statusMap[queryExpr] = prediction.StatusReady
}

func (a *aggregateSignals) GetSignals(queryExpr string) (map[string]*aggregateSignal, prediction.Status) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	if _, exists := a.signalMap[queryExpr]; !exists {
		return nil, prediction.StatusUnknown
	}

	m := map[string]*aggregateSignal{}
	for k, v := range a.signalMap[queryExpr] {
		m[k] = v
	}
	return m, a.statusMap[queryExpr]
}
