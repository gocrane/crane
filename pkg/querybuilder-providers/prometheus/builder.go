package prometheus

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/gocrane/crane/pkg/metricquery"
	"github.com/gocrane/crane/pkg/querybuilder"
)

// todo: later we change these templates to configurable like prometheus-adapter
const (
	// WorkloadCpuUsageExprTemplate is used to query workload cpu usage by promql,  param is namespace,workload-name,duration str
	WorkloadCpuUsageExprTemplate = `sum(irate(container_cpu_usage_seconds_total{container!="",image!="",container!="POD",namespace="%s",pod=~"^%s-.*$"}[%s]))`
	// WorkloadMemUsageExprTemplate is used to query workload mem usage by promql, param is namespace, workload-name
	WorkloadMemUsageExprTemplate = `sum(container_memory_working_set_bytes{container!="",image!="",container!="POD",namespace="%s",pod=~"^%s-.*$"})`

	// following is node exporter metric for node cpu/memory usage
	// NodeCpuUsageExprTemplate is used to query node cpu usage by promql,  param is node name which prometheus scrape, duration str
	NodeCpuUsageExprTemplate = `sum(count(node_cpu_seconds_total{mode="idle",instance=~"(%s)(:\\d+)?"}) by (mode, cpu)) - sum(irate(node_cpu_seconds_total{mode="idle",instance=~"(%s)(:\\d+)?"}[%s]))`
	// NodeMemUsageExprTemplate is used to query node cpu memory by promql,  param is node name, node name which prometheus scrape
	NodeMemUsageExprTemplate = `sum(node_memory_MemTotal_bytes{instance=~"(%s)(:\\d+)?"} - node_memory_MemAvailable_bytes{instance=~"(%s)(:\\d+)?"})`

	// PodCpuUsageExprTemplate is used to query pod cpu usage by promql,  param is namespace,pod, duration str
	PodCpuUsageExprTemplate = `sum(irate(container_cpu_usage_seconds_total{container!="POD",namespace="%s",pod="%s"}[%s]))`
	// PodMemUsageExprTemplate is used to query pod cpu usage by promql,  param is namespace,pod
	PodMemUsageExprTemplate = `sum(container_memory_working_set_bytes{container!="POD",namespace="%s",pod="%s"})`

	// ContainerCpuUsageExprTemplate is used to query container cpu usage by promql,  param is namespace,pod,container duration str
	ContainerCpuUsageExprTemplate = `irate(container_cpu_usage_seconds_total{container!="POD",namespace="%s",pod=~"^%s.*$",container="%s"}[%s])`
	// ContainerMemUsageExprTemplate is used to query container cpu usage by promql,  param is namespace,pod,container
	ContainerMemUsageExprTemplate = `container_memory_working_set_bytes{container!="POD",namespace="%s",pod=~"^%s.*$",container="%s"}`
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
			Query: fmt.Sprintf(WorkloadCpuUsageExprTemplate, metric.Workload.Namespace, metric.Workload.Name, "3m"),
		}), nil
	case v1.ResourceMemory.String():
		return promQuery(&metricquery.PrometheusQuery{
			Query: fmt.Sprintf(WorkloadMemUsageExprTemplate, metric.Workload.Namespace, metric.Workload.Name),
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
			Query: fmt.Sprintf(ContainerCpuUsageExprTemplate, metric.Container.Namespace, metric.Container.PodName, metric.Container.ContainerName, "3m"),
		}), nil
	case v1.ResourceMemory.String():
		return promQuery(&metricquery.PrometheusQuery{
			Query: fmt.Sprintf(ContainerMemUsageExprTemplate, metric.Container.Namespace, metric.Container.PodName, metric.Container.ContainerName),
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
			Query: fmt.Sprintf(PodCpuUsageExprTemplate, metric.Pod.Namespace, metric.Pod.Name, "3m"),
		}), nil
	case v1.ResourceMemory.String():
		return promQuery(&metricquery.PrometheusQuery{
			Query: fmt.Sprintf(PodMemUsageExprTemplate, metric.Pod.Namespace, metric.Pod.Name),
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
			Query: fmt.Sprintf(NodeCpuUsageExprTemplate, metric.Node.Name, metric.Node.Name, "3m"),
		}), nil
	case v1.ResourceMemory.String():
		return promQuery(&metricquery.PrometheusQuery{
			Query: fmt.Sprintf(NodeMemUsageExprTemplate, metric.Node.Name, metric.Node.Name),
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
