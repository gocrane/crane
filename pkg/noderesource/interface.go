package noderesource

import (
	predictionv1alpha1 "github.com/gocrane/api/pkg/generated/informers/externalversions/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/utils"
	"k8s.io/client-go/tools/record"
)

type Collector interface {
	Name() string
	Run(stop <-chan struct{}, stateChan chan struct {
		stateMap      map[string][]MetricTimeSeries
		collectorName string
	})
	GetLastState() map[string][]MetricTimeSeries
}

type CollectContext struct {
	TimeSeriesPredictionInformer predictionv1alpha1.TimeSeriesPredictionInformer
	Recorder                     record.EventRecorder
	NodeName                     string
	CpuStateProvider             *utils.CpuStateProvider
	TspName                      string
}
