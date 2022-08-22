package prometheus

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"

	"github.com/gocrane/crane/pkg/metricquery"
	"github.com/gocrane/crane/pkg/querybuilder"
	"github.com/gocrane/crane/pkg/utils"
	"k8s.io/apimachinery/pkg/util/sets"
)

var supportedResources = sets.NewString(v1.ResourceCPU.String(), v1.ResourceMemory.String())

var _ querybuilder.Builder = &builder{}

type builder struct {
	metric *metricquery.Metric
}

func NewPromQueryBuilder(metric *metricquery.Metric) querybuilder.Builder {
	return &builder{
		metric: metric,
	}
}

func (b *builder) BuildQuery() (*metricquery.Query, error) {
	switch b.metric.Type {
	case metricquery.WorkloadMetricType:
		return b.workloadQuery(b.metric)
	case metricquery.PodMetricType:
		return b.podQuery(b.metric)
	case metricquery.ContainerMetricType:
		return b.containerQuery(b.metric)
	case metricquery.NodeMetricType:
		return b.nodeQuery(b.metric)
	case metricquery.PromQLMetricType:
		return b.promQuery(b.metric)
	default:
		return nil, fmt.Errorf("metric type %v not supported", b.metric.Type)
	}
}

func (b *builder) workloadQuery(metric *metricquery.Metric) (*metricquery.Query, error) {
	if metric.Workload == nil {
		return nil, fmt.Errorf("metric type %v, but no WorkloadNamerInfo provided", metric.Type)
	}
	switch strings.ToLower(metric.MetricName) {
	case v1.ResourceCPU.String():
		return promQuery(&metricquery.PrometheusQuery{
			Query: utils.GetWorkloadCpuUsageExpression(metric.Workload.Namespace, metric.Workload.Name),
		}), nil
	case v1.ResourceMemory.String():
		return promQuery(&metricquery.PrometheusQuery{
			Query: utils.GetWorkloadMemUsageExpression(metric.Workload.Namespace, metric.Workload.Name),
		}), nil
	default:
		return nil, fmt.Errorf("metric type %v do not support resource metric %v. only support %v now", metric.Type, metric.MetricName, supportedResources.List())
	}
}

func (b *builder) containerQuery(metric *metricquery.Metric) (*metricquery.Query, error) {
	if metric.Container == nil {
		return nil, fmt.Errorf("metric type %v, but no ContainerNamerInfo provided", metric.Type)
	}
	switch strings.ToLower(metric.MetricName) {
	case v1.ResourceCPU.String():
		return promQuery(&metricquery.PrometheusQuery{
			Query: utils.GetContainerCpuUsageExpression(metric.Container.Namespace, metric.Container.WorkloadName, metric.Container.Name),
		}), nil
	case v1.ResourceMemory.String():
		return promQuery(&metricquery.PrometheusQuery{
			Query: utils.GetContainerMemUsageExpression(metric.Container.Namespace, metric.Container.WorkloadName, metric.Container.Name),
		}), nil
	default:
		return nil, fmt.Errorf("metric type %v do not support resource metric %v. only support %v now", metric.Type, metric.MetricName, supportedResources.List())
	}
}

func (b *builder) podQuery(metric *metricquery.Metric) (*metricquery.Query, error) {
	if metric.Pod == nil {
		return nil, fmt.Errorf("metric type %v, but no PodNamerInfo provided", metric.Type)
	}
	switch strings.ToLower(metric.MetricName) {
	case v1.ResourceCPU.String():
		return promQuery(&metricquery.PrometheusQuery{
			Query: utils.GetPodCpuUsageExpression(metric.Pod.Namespace, metric.Pod.Name),
		}), nil
	case v1.ResourceMemory.String():
		return promQuery(&metricquery.PrometheusQuery{
			Query: utils.GetPodMemUsageExpression(metric.Pod.Namespace, metric.Pod.Name),
		}), nil
	default:
		return nil, fmt.Errorf("metric type %v do not support resource metric %v. only support %v now", metric.Type, metric.MetricName, supportedResources.List())
	}
}

func (b *builder) nodeQuery(metric *metricquery.Metric) (*metricquery.Query, error) {
	if metric.Node == nil {
		return nil, fmt.Errorf("metric type %v, but no NodeNamerInfo provided", metric.Type)
	}
	switch strings.ToLower(metric.MetricName) {
	case v1.ResourceCPU.String():
		return promQuery(&metricquery.PrometheusQuery{
			Query: utils.GetNodeCpuUsageExpression(metric.Node.Name),
		}), nil
	case v1.ResourceMemory.String():
		return promQuery(&metricquery.PrometheusQuery{
			Query: utils.GetNodeMemUsageExpression(metric.Node.Name),
		}), nil
	default:
		return nil, fmt.Errorf("metric type %v do not support resource metric %v. only support %v now", metric.Type, metric.MetricName, supportedResources.List())
	}
}

func (b *builder) promQuery(metric *metricquery.Metric) (*metricquery.Query, error) {
	if metric.Prom == nil {
		return nil, fmt.Errorf("metric type %v, but no PromNamerInfo provided", metric.Type)
	}
	return promQuery(&metricquery.PrometheusQuery{
		Query: metric.Prom.QueryExpr,
	}), nil
}

func promQuery(prom *metricquery.PrometheusQuery) *metricquery.Query {
	return &metricquery.Query{
		Type:       metricquery.PrometheusMetricSource,
		Prometheus: prom,
	}
}

func init() {
	querybuilder.RegisterBuilderFactory(metricquery.PrometheusMetricSource, NewPromQueryBuilder)
}
