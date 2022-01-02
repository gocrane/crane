package metricsserver

import (
	"sync"

	"k8s.io/klog/v2"
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

func (m *MetricsServer) Collect() {
	klog.V(4).Infof("Metrics server collecting")
	return
}
