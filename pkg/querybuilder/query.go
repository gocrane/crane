package querybuilder

import (
	"sync"

	"github.com/gocrane/crane/pkg/metricquery"
)

type BuildQueryBehavior struct {
	// FederatedClusterScope means this query data source supports multiple clusters data query.
	// false means do not need use cluster as query param.
	// true means the data source maybe has multiple clusters, so must require cluster param. it will inject cluster param to the query when build query
	FederatedClusterScope bool
	// used to distiguish clusters. such as clusterid=cls-xxx
	ClusterLabelName  string
	ClusterLabelValue string
}

// Builder is an interface which is used to build query for different data sources according a context info about the query.
type Builder interface {
	BuildQuery(behavior BuildQueryBehavior) (*metricquery.Query, error)
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
