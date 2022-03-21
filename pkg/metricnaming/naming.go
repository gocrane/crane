package metricnaming

import (
	"github.com/gocrane/crane/pkg/metricquery"
	"github.com/gocrane/crane/pkg/querybuilder"
	msbuilder "github.com/gocrane/crane/pkg/querybuilder-providers/metricserver"
	prombuilder "github.com/gocrane/crane/pkg/querybuilder-providers/prometheus"
)

// MetricNamer is an interface. it is the bridge between predictor and different data sources and other component.
type MetricNamer interface {
	// Used for datasource provider, data source provider call QueryBuilder
	QueryBuilder() querybuilder.QueryBuilder
	// Used for predictor now
	BuildUniqueKey() string

	Validate() error
}

type GeneralMetricNamer struct {
	Metric *metricquery.Metric
}

func (gmn *GeneralMetricNamer) QueryBuilder() querybuilder.QueryBuilder {
	return NewQueryBuilder(gmn.Metric)
}

func (gmn *GeneralMetricNamer) BuildUniqueKey() string {
	return gmn.Metric.BuildUniqueKey()
}

func (gmn *GeneralMetricNamer) Validate() error {
	return gmn.Metric.ValidateMetric()
}

type queryBuilderFactory struct {
	metric *metricquery.Metric
}

func (q queryBuilderFactory) Builder(source metricquery.MetricSource) querybuilder.Builder {
	switch source {
	case metricquery.MetricServerMetricSource:
		return msbuilder.NewMetricServerQueryBuilder(q.metric)
	case metricquery.PrometheusMetricSource:
		return prombuilder.NewPromQueryBuilder(q.metric)
	}
	return prombuilder.NewPromQueryBuilder(q.metric)
}

func NewQueryBuilder(metric *metricquery.Metric) querybuilder.QueryBuilder {
	return &queryBuilderFactory{
		metric: metric,
	}
}
