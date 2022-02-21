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
	withCh           chan string
	delCh            chan string
	mu               sync.Mutex
}

func NewGenericPrediction(withCh, stopCh chan string) GenericPrediction {
	return GenericPrediction{
		withCh:     withCh,
		delCh:      stopCh,
		mu:         sync.Mutex{},
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

func (p *GenericPrediction) WithQuery(queryExpr string) error {
	if queryExpr == "" {
		return fmt.Errorf("empty query expression")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.querySet[queryExpr]; !exists {
		p.querySet[queryExpr] = struct{}{}
		p.withCh <- queryExpr
	}

	return nil
}

func (p *GenericPrediction) DeleteQuery(queryExpr string) error {
	if queryExpr == "" {
		return fmt.Errorf("empty query expression")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.querySet[queryExpr]; exists {
		delete(p.querySet, queryExpr)
		p.delCh <- queryExpr
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
