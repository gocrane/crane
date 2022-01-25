package metricsserver

import (
	"sync"
)

type MetricsServer struct {
	Name        string
	StatusCache sync.Map
}

func NewMetricsServer() *MetricsServer {
	m := MetricsServer{
		Name:        "metrics-server",
		StatusCache: sync.Map{},
	}
	return &m
}

func (m *MetricsServer) GetName() string {
	return m.Name
}

func (m *MetricsServer) Collect() {
	return
}
