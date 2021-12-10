package metricsserver

import (
	"fmt"
	"sync"
)

type MetricsServer struct {
	Name        string
	StatusCache sync.Map
}

func NewMetricsServer() *MetricsServer {
	m := MetricsServer{
		Name:        "metricsserver",
		StatusCache: sync.Map{},
	}
	return &m
}

func (m *MetricsServer) GetName() string {
	return m.Name
}

func (e *MetricsServer) Collect() {
	fmt.Println("metrics server collecting")
}
