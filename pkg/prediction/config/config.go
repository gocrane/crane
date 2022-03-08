package config

import (
	"fmt"
	"sort"
	"strings"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
)

const (
	// WorkloadCpuUsagePromQLFmtStr is used to query workload cpu usage by promql,  param is namespace,workload-name,duration str
	WorkloadCpuUsagePromQLFmtStr = `sum (irate (container_cpu_usage_seconds_total{container!="",image!="",container!="POD",namespace="%s",pod=~"^%s-.*$"}[%s]))`
	// WorkloadMemUsagePromQLFmtStr is used to query workload mem usage by promql, param is namespace, workload-name
	WorkloadMemUsagePromQLFmtStr = `sum(container_memory_working_set_bytes{container!="",image!="",container!="POD",namespace="%s",pod=~"^%s-.*$"})`

	// following is node exporter metric for node cpu/memory usage
	// NodeCpuUsagePromQLFmtStr is used to query node cpu usage by promql,  param is node name which prometheus scrape, duration str
	NodeCpuUsagePromQLFmtStr = `sum(count(node_cpu_seconds_total{mode="idle",instance=~"(%s)(:\\d+)?"}) by (mode, cpu)) - sum(irate(node_cpu_seconds_total{mode="idle",instance=~"(%s)(:\\d+)?"}[%s]))`
	// NodeMemUsagePromQLFmtStr is used to query node cpu memory by promql,  param is node name, node name which prometheus scrape
	NodeMemUsagePromQLFmtStr = `sum(node_memory_MemTotal_bytes{instance=~"(%s)(:\\d+)?"} - node_memory_MemAvailable_bytes{instance=~"(%s)(:\\d+)?"})`
)

//var UpdateEventBroadcaster Broadcaster = NewBroadcaster()
//var DeleteEventBroadcaster Broadcaster = NewBroadcaster()

const TargetKindNode = "Node"

func metricSelectorToQueryExpr(m *predictionapi.MetricQuery) string {
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
