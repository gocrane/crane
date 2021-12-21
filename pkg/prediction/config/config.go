package config

import (
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/utils/log"
)

const (
	// WorkloadCpuUsagePromQLFmtStr is used to query workload cpu usage by promql,  param is namespace,workload-name,duration str
	WorkloadCpuUsagePromQLFmtStr = `sum (irate (container_cpu_usage_seconds_total{container!="",image!="",name=~"^k8s_.*",container!="POD",namespace="%s",pod=~"^%s-.*$"}[%s]))`
	// WorkloadMemUsagePromQLFmtStr is used to query workload mem usage by promql, param is namespace, workload-name
	WorkloadMemUsagePromQLFmtStr = `sum(container_memory_working_set_bytes{container!="",image!="", name=~"^k8s_.*",container!="POD",namespace="%s",pod=~"^%s-.*$"})`

	// following is node exporter and cadvisor metric for node cpu/memory usage
	// NodeCpuUsagePromQLFmtStr is used to query node cpu usage by promql,  param is node name which prometheus scrape, duration str
	NodeCpuUsagePromQLFmtStr = `sum(machine_cpu_cores{instance=~"%s.*"}) - sum(irate(node_cpu_seconds_total{mode="idle",instance=~"%s.*"}[%s]))`
	// NodeMemUsagePromQLFmtStr is used to query node cpu memory by promql,  param is node name, node name which prometheus scrape
	NodeMemUsagePromQLFmtStr = `sum(node_memory_MemTotal_bytes{instance=~"^%s.*"} - node_memory_MemAvailable_bytes{instance=~"^%s.*"})`
)

var UpdateEventBroadcaster Broadcaster = NewBroadcaster()
var DeleteEventBroadcaster Broadcaster = NewBroadcaster()

var logger = log.Logger()

func (c *MetricContext) WithApiConfig(conf *v1alpha1.PredictionMetric) {
	if conf.MetricQuery != nil {
		logger.V(2).Info("WithApiConfig", "metricSelector", metricSelectorToQueryExpr(conf.MetricQuery))
	}
	if conf.ExpressionQuery != nil {
		logger.V(2).Info("WithApiConfig", "queryExpr", conf.ExpressionQuery.Expression)
	}
	if conf.ResourceQuery != nil {
		logger.V(2).Info("WithApiConfig", "resourceQuery", conf.ResourceQuery)
	}

	UpdateEventBroadcaster.Write(c.ConvertApiMetric2InternalConfig(conf))
}

const TargetKindNode = "Node"

type MetricContext struct {
	Namespace  string
	TargetKind string
	Name       string
}

func (c *MetricContext) WithApiConfigs(configs []v1alpha1.PredictionMetric) {
	for _, conf := range configs {
		c.WithApiConfig(&conf)
	}
}

func (c *MetricContext) DeleteApiConfig(conf *v1alpha1.PredictionMetric) {
	if conf.MetricQuery != nil {
		logger.V(2).Info("DeleteApiConfig", "metricSelector", metricSelectorToQueryExpr(conf.MetricQuery))
	} else if conf.ExpressionQuery != nil {
		logger.V(2).Info("DeleteApiConfig", "queryExpr", conf.ExpressionQuery.Expression)
	}
	DeleteEventBroadcaster.Write(c.ConvertApiMetric2InternalConfig(conf))
}

func (c *MetricContext) DeleteApiConfigs(configs []v1alpha1.PredictionMetric) {
	for _, conf := range configs {
		c.DeleteApiConfig(&conf)
	}
}

func (c *MetricContext) WithConfigs(configs []*Config) {
	for _, conf := range configs {
		c.WithConfig(conf)
	}
}

func (c *MetricContext) WithConfig(conf *Config) {
	if conf.Metric != nil {
		logger.V(2).Info("WithConfig", "metricSelector", metricSelectorToQueryExpr(conf.Metric))
	} else if conf.Expression != nil {
		logger.V(2).Info("WithConfig", "queryExpr", conf.Expression)
	}
	UpdateEventBroadcaster.Write(conf)
}

func (c *MetricContext) DeleteConfig(conf *Config) {
	if conf.Metric != nil {
		logger.V(2).Info("DeleteConfig", "metricSelector", metricSelectorToQueryExpr(conf.Metric))
	} else if conf.Expression != nil {
		logger.V(2).Info("DeleteConfig", "queryExpr", conf.Expression.Expression)
	}
	DeleteEventBroadcaster.Write(conf)
}

func metricSelectorToQueryExpr(m *v1alpha1.MetricQuery) string {
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

func (c *MetricContext) ResourceToPromQueryExpr(resourceName *corev1.ResourceName) string {
	if strings.ToLower(c.TargetKind) == strings.ToLower(TargetKindNode) {
		switch *resourceName {
		case corev1.ResourceCPU:
			return fmt.Sprintf(NodeCpuUsagePromQLFmtStr, c.Name, c.Name, "1m")
		case corev1.ResourceMemory:
			return fmt.Sprintf(NodeMemUsagePromQLFmtStr, c.Name, c.Name)
		}
	} else {
		switch *resourceName {
		case corev1.ResourceCPU:
			return fmt.Sprintf(WorkloadCpuUsagePromQLFmtStr, c.Namespace, c.Name, "1m")
		case corev1.ResourceMemory:
			return fmt.Sprintf(WorkloadMemUsagePromQLFmtStr, c.Namespace, c.Name)
		}
	}
	return ""
}
