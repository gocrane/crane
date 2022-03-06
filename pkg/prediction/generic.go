package prediction

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
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
	historyProvider  providers.Interface
	realtimeProvider providers.Interface
	metricsMap       map[string][]common.QueryCondition
	querySet         map[string]struct{}
	WithCh           chan QueryExprWithCaller
	DelCh            chan QueryExprWithCaller
	mutex            sync.Mutex
}

func NewGenericPrediction(withCh, delCh chan QueryExprWithCaller) GenericPrediction {
	return GenericPrediction{
		WithCh:     withCh,
		DelCh:      delCh,
		mutex:      sync.Mutex{},
		metricsMap: map[string][]common.QueryCondition{},
		querySet:   map[string]struct{}{},
	}
}

func (p *GenericPrediction) GetHistoryProvider() providers.Interface {
	return p.historyProvider
}

func (p *GenericPrediction) GetRealtimeProvider() providers.Interface {
	return p.realtimeProvider
}

func (p *GenericPrediction) WithProviders(providers map[string]providers.Interface) {
	for k, v := range providers {
		if k == HistoryProvider {
			p.historyProvider = v
		} else if k == RealtimeProvider {
			p.realtimeProvider = v
		}
	}
}

func (p *GenericPrediction) WithQuery(queryExpr string, caller string, config config.Config) error {
	if queryExpr == "" {
		return fmt.Errorf("empty query expression")
	}
	if caller == "" {
		return fmt.Errorf("empty caller")
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	q := QueryExprWithCaller{
		QueryExpr: queryExpr,
		Caller:    caller,
		Config:    config,
	}

	if _, exists := p.querySet[q.String()]; !exists {
		p.querySet[q.String()] = struct{}{}
		klog.V(4).InfoS("Put tuple{query,caller,config} into with channel.", "query", q.QueryExpr, "caller", q.Caller)
		p.WithCh <- q
	}

	return nil
}

func (p *GenericPrediction) DeleteQuery(queryExpr string, caller string) error {
	if queryExpr == "" {
		return fmt.Errorf("empty query expression")
	}
	if caller == "" {
		return fmt.Errorf("empty caller")
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	q := QueryExprWithCaller{
		QueryExpr: queryExpr,
		Caller:    caller,
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
	QueryExpr string
	Config    config.Config
	Caller    string
}

func (q QueryExprWithCaller) String() string {
	return fmt.Sprintf("%s####%s", q.Caller, q.QueryExpr)
}
