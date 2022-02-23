package prediction

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/providers"
)

const (
	HistoryProvider  = "__history"
	RealtimeProvider = "__realtime"
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
	withCh           chan QueryExprWithCaller
	delCh            chan QueryExprWithCaller
	mu               sync.Mutex
}

func NewGenericPrediction(withCh, stopCh chan QueryExprWithCaller) GenericPrediction {
	return GenericPrediction{
		withCh:     withCh,
		delCh:      stopCh,
		mu:         sync.Mutex{},
		metricsMap: map[string][]common.QueryCondition{},
		querySet:   map[string]struct{}{},
	}
}

type QueryExprWithCaller struct {
	QueryExpr string
	Caller    string
}

func (q QueryExprWithCaller) String() string {
	return fmt.Sprintf("%s####%s", q.Caller, q.QueryExpr)
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

func (p *GenericPrediction) WithQuery(queryExpr string, caller string) error {
	if queryExpr == "" {
		return fmt.Errorf("empty query expression")
	}
	if caller == "" {
		return fmt.Errorf("empty caller")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	q := QueryExprWithCaller{
		QueryExpr: queryExpr,
		Caller:    caller,
	}

	if _, exists := p.querySet[q.String()]; !exists {
		p.querySet[q.String()] = struct{}{}
		p.withCh <- q
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

	p.mu.Lock()
	defer p.mu.Unlock()

	q := QueryExprWithCaller{
		QueryExpr: queryExpr,
		Caller:    caller,
	}

	if _, exists := p.querySet[q.String()]; exists {
		delete(p.querySet, q.String())
		p.delCh <- q
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
