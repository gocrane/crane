package grpc

import (
	"github.com/gocrane/crane/pkg/metricquery"
	"github.com/gocrane/crane/pkg/querybuilder"
)

var _ querybuilder.Builder = &builder{}

type builder struct {
	metric *metricquery.Metric
}

func NewQueryBuilder(metric *metricquery.Metric) querybuilder.Builder {
	return &builder{
		metric: metric,
	}
}

func (b builder) BuildQuery(behavior querybuilder.BuildQueryBehavior) (*metricquery.Query, error) {
	return gRPCQuery(&metricquery.GenericQuery{Metric: b.metric}), nil
}

func gRPCQuery(query *metricquery.GenericQuery) *metricquery.Query {
	return &metricquery.Query{
		Type:         metricquery.GrpcMetricSource,
		GenericQuery: query,
	}
}

func init() {
	querybuilder.RegisterBuilderFactory(metricquery.GrpcMetricSource, NewQueryBuilder)
}
