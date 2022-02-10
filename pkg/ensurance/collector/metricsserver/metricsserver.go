package metricsserver

import (
	"sync"

	"github.com/gocrane/crane/pkg/common"

	"github.com/gocrane/crane/pkg/ensurance/collector/types"
)

type MetricsServer struct {
	name        types.CollectType
	statusCache sync.Map
}

func NewMetricsServer() *MetricsServer {
	m := MetricsServer{
		name:        types.MetricsServerCollectorType,
		statusCache: sync.Map{},
	}
	return &m
}

func (m *MetricsServer) GetType() types.CollectType {
	return m.name
}

func (m *MetricsServer) Collect() (map[string][]common.TimeSeries, error) {
	return nil, nil
}

func (m *MetricsServer) Stop() error {
	return nil
}
