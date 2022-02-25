package prediction

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/prediction/config"
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
	historyProvider      providers.Interface
	realtimeProvider     providers.Interface
	metricsMap           map[string][]common.QueryCondition
	querySet             map[string]struct{}
	withQueryBroadcaster config.Broadcaster
	mu                   sync.Mutex
}

func NewGenericPrediction(withQueryBroadcaster config.Broadcaster) GenericPrediction {
	return GenericPrediction{
		withQueryBroadcaster: withQueryBroadcaster,
		mu:                   sync.Mutex{},
		metricsMap:           map[string][]common.QueryCondition{},
		querySet:             map[string]struct{}{},
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

func (p *GenericPrediction) WithQuery(query string) error {
	if query == "" {
		return fmt.Errorf("empty query")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.querySet[query]; !exists {
		p.querySet[query] = struct{}{}
		p.withQueryBroadcaster.Write(query)
	}

	return nil
}

func AggregateSignalKey(id string, labels []common.Label) string {
	labelSet := make([]string, 0, len(labels))
	for _, label := range labels {
		labelSet = append(labelSet, label.Name+"="+label.Value)
	}
	sort.Strings(labelSet)
	return id + "#" + strings.Join(labelSet, ",")
}
