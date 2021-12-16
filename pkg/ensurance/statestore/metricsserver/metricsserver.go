package metricsserver

import (
	"sync"

	"github.com/gocrane/crane/pkg/utils/log"
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
	log.Logger().V(4).Info("Metrics server collecting")
	return
}
