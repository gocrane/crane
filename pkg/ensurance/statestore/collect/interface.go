package collect

import "sync"

type Collector interface {
	GetName() string
	Collect()
	List() sync.Map
	AddMetric(metricName string)
	DeleteMetric(metricName string)
}
