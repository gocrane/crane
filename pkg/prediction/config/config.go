package config

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/utils/log"
)

const (
	// WorkloadCpuUsagePromQLFmtStr is used to query workload cpu usage by promql,  param is namespace,workload-name,duration str
	WorkloadCpuUsagePromQLFmtStr = `sum (irate (container_cpu_usage_seconds_total{container!="",image!="",name=~"^k8s_.*",container!="POD",namespace="%s",pod=~"^%s-.*$"}[%s]))`
	// WorkloadMemUsagePromQLFmtStr is used to query workload mem usage by promql, param is namespace, workload-name
	WorkloadMemUsagePromQLFmtStr = `sum(container_memory_working_set_bytes{container!="",image!="", name=~"^k8s_.*",container!="POD",namespace="%s",pod=~"^%s-.*$"})`

	// NodeCpuUsagePromQLFmtStr is used to query node cpu usage by promql,  param is node name which prometheus scrape, duration str
	NodeCpuUsagePromQLFmtStr = `1-avg(rate(node_cpu_seconds_total{mode="idle",instance=~"^%s.*"}[%s]))`
	// NodeMemUsagePromQLFmtStr is used to query node cpu memory by promql,  param is node name, node name which prometheus scrape
	NodeMemUsagePromQLFmtStr = `sum(node_memory_MemTotal_bytes{instance=~"^%s.*"} - node_memory_MemAvailable_bytes{instance=~"^%s.*"})`
)

var UpdateEventBroadcaster Broadcaster = NewBroadcaster()
var DeleteEventBroadcaster Broadcaster = NewBroadcaster()

var logger = log.Logger()

func WithApiConfig(conf *v1alpha1.PredictionMetric) {
	if conf.MetricSelector != nil {
		logger.V(2).Info("WithApiConfig", "metricSelector", metricSelectorToQueryExpr(conf.MetricSelector))
	} else if conf.Query != nil {
		logger.V(2).Info("WithApiConfig", "queryExpr", conf.Query.Expression)
	}

	UpdateEventBroadcaster.Write(ConvertApiMetric2InternalConfig(conf))
}

func WithApiConfigs(configs []v1alpha1.PredictionMetric) {
	for _, conf := range configs {
		WithApiConfig(&conf)
	}
}

func DeleteApiConfig(conf *v1alpha1.PredictionMetric) {
	if conf.MetricSelector != nil {
		logger.V(2).Info("DeleteApiConfig", "metricSelector", metricSelectorToQueryExpr(conf.MetricSelector))
	} else if conf.Query != nil {
		logger.V(2).Info("DeleteApiConfig", "queryExpr", conf.Query.Expression)
	}
	DeleteEventBroadcaster.Write(ConvertApiMetric2InternalConfig(conf))
}

func DeleteApiConfigs(configs []v1alpha1.PredictionMetric) {
	for _, conf := range configs {
		DeleteApiConfig(&conf)
	}
}

func WithConfigs(configs []*Config) {
	for _, conf := range configs {
		WithConfig(conf)
	}
}

func WithConfig(conf *Config) {
	if conf.MetricSelector != nil {
		logger.V(2).Info("WithConfig", "metricSelector", metricSelectorToQueryExpr(conf.MetricSelector))
	} else if conf.Query != nil {
		logger.V(2).Info("WithConfig", "queryExpr", conf.Query.Expression)
	}
	UpdateEventBroadcaster.Write(conf)
}

func DeleteConfig(conf *Config) {
	if conf.MetricSelector != nil {
		logger.V(2).Info("DeleteConfig", "metricSelector", metricSelectorToQueryExpr(conf.MetricSelector))
	} else if conf.Query != nil {
		logger.V(2).Info("DeleteConfig", "queryExpr", conf.Query.Expression)
	}
	DeleteEventBroadcaster.Write(conf)
}

func metricSelectorToQueryExpr(m *v1alpha1.MetricSelector) string {
	conditions := make([]string, 0, len(m.QueryConditions))
	for _, cond := range m.QueryConditions {
		values := make([]string, 0, len(cond.Value))
		for _, val := range cond.Value {
			values = append(values, val)
		}
		sort.Strings(values)
		conditions = append(conditions, fmt.Sprintf("%s%s[%s]", cond.Key, cond.Operator, strings.Join(values, ",")))
	}
	sort.Strings(conditions)
	return fmt.Sprintf("%s{%s}", m.MetricName, strings.Join(conditions, ","))
}

func WorkloadResourceToPromQueryExpr(resourceMetric *v1alpha1.WorkloadResource) string {
	switch resourceMetric.Resource {
	case v1alpha1.ResourceCPU:
		return fmt.Sprintf(WorkloadCpuUsagePromQLFmtStr, resourceMetric.Namespace, resourceMetric.Name, "1m")
	case v1alpha1.ResourceMemory:
		return fmt.Sprintf(WorkloadMemUsagePromQLFmtStr, resourceMetric.Namespace, resourceMetric.Name)
	}
	return ""
}

func NodeResourceToPromQueryExpr(resourceMetric *v1alpha1.NodeResource) string {
	switch resourceMetric.Resource {
	case v1alpha1.ResourceCPU:
		return fmt.Sprintf(NodeCpuUsagePromQLFmtStr, resourceMetric.Name, "1m")
	case v1alpha1.ResourceMemory:
		return fmt.Sprintf(NodeMemUsagePromQLFmtStr, resourceMetric.Name, resourceMetric.Name)
	}
	return ""
}

// todo
func WorkloadResourceToMetricSelector(resourceMetric *v1alpha1.WorkloadResource) *v1alpha1.MetricSelector {
	return nil
}
