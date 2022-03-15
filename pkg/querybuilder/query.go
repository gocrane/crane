package querybuilder

import (
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
