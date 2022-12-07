package prometheus

import (
	"fmt"
	"github.com/gocrane/crane/pkg/prometheus-adapter"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/gocrane/crane/pkg/metricquery"
	"github.com/gocrane/crane/pkg/querybuilder"
	"github.com/gocrane/crane/pkg/utils"
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

func (b *builder) workloadQuery(metric *metricquery.Metric) (query *metricquery.Query, err error) {
	if metric.Workload == nil {
		return nil, fmt.Errorf("metric type %v, but no WorkloadNamerInfo provided", metric.Type)
	}
	mrs := prometheus_adapter.GetMetricRules()
	var metricRule *prometheus_adapter.MetricRule
	var queryExpr string

	switch strings.ToLower(metric.MetricName) {
	case v1.ResourceCPU.String():
		metricRule = prometheus_adapter.MatchMetricRule(mrs.MetricRulesExternal, prometheus_adapter.WorkloadCpuUsageExpression)
		if metricRule == nil {
			queryExpr = utils.GetWorkloadCpuUsageExpression(metric.Workload.Namespace, metric.Workload.Name, "")
		}
	case v1.ResourceMemory.String():
		metricRule = prometheus_adapter.MatchMetricRule(mrs.MetricRulesExternal, prometheus_adapter.WorkloadMemUsageExpression)
		if metricRule == nil {
			queryExpr = utils.GetWorkloadMemUsageExpression(metric.Workload.Namespace, metric.Workload.Name, "")
		}
	default:
		return nil, fmt.Errorf("metric type %v do not support resource metric %v. only support %v now", metric.Type, metric.MetricName, supportedResources.List())
	}

	if queryExpr == "" {
		queryExpr, err = metricRule.QueryForSeries([]string{fmt.Sprintf("namespace=\"%s\"", metric.Workload.Namespace), fmt.Sprintf("pod=~\"%s\"", utils.GetPodNameReg(metric.Workload.Name, metric.Workload.Kind))})
	}
	return promQuery(&metricquery.PrometheusQuery{Query: queryExpr}), err
}

func (b *builder) nodeQuery(metric *metricquery.Metric) (query *metricquery.Query, err error) {
	if metric.Node == nil {
		return nil, fmt.Errorf("metric type %v, but no NodeNamerInfo provided", metric.Type)
	}
	mrs := prometheus_adapter.GetMetricRules()
	var metricRule *prometheus_adapter.MetricRule
	var queryExpr string

	switch strings.ToLower(metric.MetricName) {
	case v1.ResourceCPU.String():
		metricRule = prometheus_adapter.MatchMetricRule(mrs.MetricRulesExternal, prometheus_adapter.NodeCpuUsageExpression)
		if metricRule == nil {
			queryExpr = utils.GetNodeCpuUsageExpression(metric.Node.Name)
		}
	case v1.ResourceMemory.String():
		metricRule = prometheus_adapter.MatchMetricRule(mrs.MetricRulesExternal, prometheus_adapter.NodeMemUsageExpression)
		if metricRule == nil {
			queryExpr = utils.GetNodeMemUsageExpression(metric.Node.Name)
		}
	default:
		return nil, fmt.Errorf("metric type %v do not support resource metric %v. only support %v now", metric.Type, metric.MetricName, supportedResources.List())
	}

	if queryExpr == "" {
		queryExpr, err = metricRule.QueryForSeries([]string{fmt.Sprintf("instance=~\"(%s)(:\\d+)?\"", metric.Node.Name)})
	}

	return promQuery(&metricquery.PrometheusQuery{Query: queryExpr}), err
}

func (b *builder) containerQuery(metric *metricquery.Metric) (query *metricquery.Query, err error) {
	if metric.Container == nil {
		return nil, fmt.Errorf("metric type %v, but no ContainerNamerInfo provided", metric.Type)
	}
	mrs := prometheus_adapter.GetMetricRules()
	var metricRule *prometheus_adapter.MetricRule
	var queryExpr string

	switch strings.ToLower(metric.MetricName) {
	case v1.ResourceCPU.String():
		metricRule = prometheus_adapter.MatchMetricRule(mrs.MetricRulesExternal, prometheus_adapter.ContainerCpuUsageExpression)
		if metricRule == nil {
			queryExpr = utils.GetContainerCpuUsageExpression(metric.Container.Namespace, metric.Container.WorkloadName, "", metric.Container.Name)
		}
	case v1.ResourceMemory.String():
		metricRule = prometheus_adapter.MatchMetricRule(mrs.MetricRulesExternal, prometheus_adapter.ContainerMemUsageExpression)
		if metricRule == nil {
			queryExpr = utils.GetContainerMemUsageExpression(metric.Container.Namespace, metric.Container.WorkloadName, "", metric.Container.Name)
		}
	default:
		return nil, fmt.Errorf("metric type %v do not support resource metric %v. only support %v now", metric.Type, metric.MetricName, supportedResources.List())
	}

	if queryExpr == "" {
		queryExpr, err = metricRule.QueryForSeries([]string{fmt.Sprintf("namespace=\"%s\"", metric.Container.Namespace), fmt.Sprintf("pod=~\"%s\"", utils.GetPodNameReg(metric.Workload.Name, metric.Workload.Kind)), fmt.Sprintf("container=\"%s\"", metric.Container.Name)})
	}
	return promQuery(&metricquery.PrometheusQuery{
		Query: queryExpr,
	}), err
}

func (b *builder) podQuery(metric *metricquery.Metric) (query *metricquery.Query, err error) {
	if metric.Pod == nil {
		return nil, fmt.Errorf("metric type %v, but no PodNamerInfo provided", metric.Type)
	}
	mrs := prometheus_adapter.GetMetricRules()
	var metricRule *prometheus_adapter.MetricRule
	var queryExpr string

	switch strings.ToLower(metric.MetricName) {
	case v1.ResourceCPU.String():
		metricRule = prometheus_adapter.MatchMetricRule(mrs.MetricRulesExternal, prometheus_adapter.PodCpuUsageExpression)
		if metricRule == nil {
			queryExpr = utils.GetPodCpuUsageExpression(metric.Pod.Namespace, metric.Pod.Name)
		}
	case v1.ResourceMemory.String():
		metricRule = prometheus_adapter.MatchMetricRule(mrs.MetricRulesExternal, prometheus_adapter.PodMemUsageExpression)
		if metricRule == nil {
			queryExpr = utils.GetPodMemUsageExpression(metric.Pod.Namespace, metric.Pod.Name)
		}
	default:
		return nil, fmt.Errorf("metric type %v do not support resource metric %v. only support %v now", metric.Type, metric.MetricName, supportedResources.List())
	}

	if queryExpr == "" {
		queryExpr, err = metricRule.QueryForSeries([]string{fmt.Sprintf("namespace=\"%s\"", metric.Pod.Namespace), fmt.Sprintf("pod=\"%s\"", metric.Pod.Name)})
	}
	return promQuery(&metricquery.PrometheusQuery{
		Query: queryExpr,
	}), err
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
