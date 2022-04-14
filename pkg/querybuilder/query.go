package querybuilder

import (
	"sync"

	"github.com/gocrane/crane/pkg/metricquery"
)

// Builder is an interface which is used to build query for different data sources according a context info about the query.
type Builder interface {
	BuildQuery() (*metricquery.Query, error)
}

// QueryBuilder is an Builder factory to make Builders
type QueryBuilder interface {
	Builder(source metricquery.MetricSource) Builder
}

var (
	factoryLock    sync.Mutex
	builderFactory = make(map[metricquery.MetricSource]BuilderFactoryFunc)
)

type BuilderFactoryFunc func(metric *metricquery.Metric) Builder

func RegisterBuilderFactory(metricSource metricquery.MetricSource, initFunc BuilderFactoryFunc) {
	factoryLock.Lock()
	defer factoryLock.Unlock()
	builderFactory[metricSource] = initFunc
}

func GetBuilderFactory(metricSource metricquery.MetricSource) BuilderFactoryFunc {
	factoryLock.Lock()
	defer factoryLock.Unlock()
	return builderFactory[metricSource]
}
