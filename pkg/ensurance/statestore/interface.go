package statestore

import (
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gocrane/crane/pkg/ensurance/manager"
)

type StateStore interface {
	manager.Manager
	List() sync.Map
	AddMetric(key string, metricName string, Selector *metav1.LabelSelector)
	DeleteMetric(key string)
}
