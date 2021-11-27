package statestore

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gocrane/crane/pkg/ensurance/manager"
	"github.com/gocrane/crane/pkg/ensurance/statestore/types"
	"github.com/gocrane/crane/pkg/utils"
)

type StateStore interface {
	manager.Manager
	List() map[string][]utils.TimeSeries
	AddMetric(key string, t types.CollectType, metricName string, Selector *metav1.LabelSelector) error
	DeleteMetric(key string, t types.CollectType)
}

type collector interface {
	GetType() types.CollectType
	Collect() (map[string][]utils.TimeSeries, error)
}
