package metricserver

import (
	"github.com/gocrane/crane/pkg/metricquery"
	"github.com/gocrane/crane/pkg/querybuilder"
)

var _ querybuilder.Builder = &builder{}

type builder struct {
	metric *metricquery.Metric
}

func NewMetricServerQueryBuilder(metric *metricquery.Metric) querybuilder.Builder {
	return &builder{
		metric: metric,
	}
}

func (b builder) BuildQuery() (*metricquery.Query, error) {
	return metricServerQuery(&metricquery.MetricServerQuery{Metric: b.metric}), nil
}

func metricServerQuery(query *metricquery.MetricServerQuery) *metricquery.Query {
	return &metricquery.Query{
		Type:         metricquery.MetricServerMetricSource,
		MetricServer: query,
	}
}

func init() {
	querybuilder.RegisterBuilderFactory(metricquery.MetricServerMetricSource, NewMetricServerQueryBuilder)
}
