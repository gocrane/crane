package prediction

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/providers"
)

const (
	HistoryProvider  = "__history"
	RealtimeProvider = "__realtime"
)

type Status string

const (
	StatusReady      Status = "Ready"
	StatusNotStarted Status = "NotStarted"
	StatusUnknown    Status = "Unknown"
	StatusDeleted    Status = "Deleted"
)

type WithMetricEvent struct {
	MetricName string
	Conditions []common.QueryCondition
}

type GenericPrediction struct {
	historyProvider  providers.History
	realtimeProvider providers.RealTime
	metricsMap       map[string][]common.QueryCondition
	querySet         map[string]struct{}
	WithCh           chan QueryExprWithCaller
	DelCh            chan QueryExprWithCaller
	mutex            sync.Mutex
}

func NewGenericPrediction(realtimeProvider providers.RealTime, historyProvider providers.History, withCh, delCh chan QueryExprWithCaller) GenericPrediction {
	return GenericPrediction{
		WithCh:           withCh,
		DelCh:            delCh,
		mutex:            sync.Mutex{},
		metricsMap:       map[string][]common.QueryCondition{},
		querySet:         map[string]struct{}{},
		realtimeProvider: realtimeProvider,
		historyProvider:  historyProvider,
	}
}

func (p *GenericPrediction) GetHistoryProvider() providers.History {
	return p.historyProvider
}

func (p *GenericPrediction) GetRealtimeProvider() providers.RealTime {
	return p.realtimeProvider
}

func (p *GenericPrediction) WithQuery(namer metricnaming.MetricNamer, caller string, config config.Config) error {
	if caller == "" {
		return fmt.Errorf("empty caller")
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	q := QueryExprWithCaller{
		MetricNamer: namer,
		Caller:      caller,
		Config:      config,
	}

	if _, exists := p.querySet[q.String()]; !exists {
		p.querySet[q.String()] = struct{}{}
		klog.V(4).InfoS("Put tuple{query,caller,config} into with channel.", "query", q.MetricNamer.BuildUniqueKey(), "caller", q.Caller)
		p.WithCh <- q
	}

	return nil
}

func (p *GenericPrediction) DeleteQuery(namer metricnaming.MetricNamer, caller string) error {
	if caller == "" {
		return fmt.Errorf("empty caller")
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	q := QueryExprWithCaller{
		MetricNamer: namer,
		Caller:      caller,
	}

	if _, exists := p.querySet[q.String()]; exists {
		delete(p.querySet, q.String())
		p.DelCh <- q
	}

	return nil
}

func AggregateSignalKey(labels []common.Label) string {
	labelSet := make([]string, 0, len(labels))
	for _, label := range labels {
		labelSet = append(labelSet, label.Name+"="+label.Value)
	}
	sort.Strings(labelSet)
	return strings.Join(labelSet, ",")
}

type QueryExprWithCaller struct {
	MetricNamer metricnaming.MetricNamer
	Config      config.Config
	Caller      string
}

func (q QueryExprWithCaller) String() string {
	return fmt.Sprintf("%s####%s", q.Caller, q.MetricNamer.BuildUniqueKey())
}
