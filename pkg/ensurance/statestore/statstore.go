package statestore

import (
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gocrane/crane/pkg/ensurance/statestore/collect"
	"github.com/gocrane/crane/pkg/utils/clogs"
)

type stateStoreManager struct {
	collectors []collect.Collector
}

func NewStateStoreManager() StateStore {
	e := collect.NewEBPF()
	n := collect.NewNodeLocal()
	m := collect.NewMetricsServer()

	collectors := []collect.Collector{e, n, m}

	return &stateStoreManager{collectors: collectors}
}

func (s *stateStoreManager) Name() string {
	return "StateStoreManager"
}

func (s *stateStoreManager) Run(stop <-chan struct{}) {
	go func() {
		updateTicker := time.NewTicker(10 * time.Second)
		defer updateTicker.Stop()
		for {
			select {
			case <-updateTicker.C:
				clogs.Log().V(2).Info("StateStore run periodically")
				for _, c := range s.collectors {
					c.Collect()
				}
			case <-stop:
				clogs.Log().V(2).Info("StateStore exit")
				return
			}
		}
	}()

	return
}

func (s *stateStoreManager) List() sync.Map {
	// for each collect to get status
	return sync.Map{}
}

func (s *stateStoreManager) AddMetric(key string, metricName string, Selector *metav1.LabelSelector) {
	return
}

func (s *stateStoreManager) DeleteMetric(key string) {
	return
}
