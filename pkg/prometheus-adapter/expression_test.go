package prometheus_adapter

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/prometheus-adapter/pkg/config"

	"testing"
)

func TestQueryForSeriesResource(t *testing.T) {
	containerQuery := `sum(rate(container_cpu_usage_seconds_total{<<.LabelMatchers>>}[3m])) by (<<.GroupBy>>)`
	cfg := &config.ResourceRules{
		CPU: config.ResourceRule{
			ContainerQuery: containerQuery,
			Resources: config.ResourceMapping{
				Overrides: map[string]config.GroupResource{},
			},
			ContainerLabel: "container",
		},
	}

	metricRules, _ := GetMetricRulesFromResourceRules(*cfg, &meta.DefaultRESTMapper{})
	expression, err := metricRules[0].QueryForSeries([]string{})
	fmt.Println(expression, err)
}

func TestQueryForSeriesRules(t *testing.T) {
	seriesQuery := `nginx_concurrent_utilization{pod_namespace!="",pod_name!=""}`
	metricsQuery := `sum(<<.Series>>{<<.LabelMatchers>>}) by (<<.GroupBy>>)`
	discoveryRule := config.DiscoveryRule{
		SeriesQuery:  seriesQuery,
		MetricsQuery: metricsQuery,
	}

	metricRules, _ := GetMetricRulesFromDiscoveryRule([]config.DiscoveryRule{discoveryRule}, &meta.DefaultRESTMapper{})
	expression, err := metricRules[0].QueryForSeries([]string{})
	fmt.Println(expression, err)
}

func TestQueryForSeriesExternalRules(t *testing.T) {
	seriesQuery := `nginx_concurrent_utilization{pod_namespace!="",pod_name!=""}`
	metricsQuery := `sum by(node, route)(rate(<<.Series>>{<<.LabelMatchers>>}[1m]))`
	discoveryRule := config.DiscoveryRule{
		SeriesQuery:  seriesQuery,
		MetricsQuery: metricsQuery,
	}

	metricRules, _ := GetMetricRulesFromDiscoveryRule([]config.DiscoveryRule{discoveryRule}, &meta.DefaultRESTMapper{})
	expression, err := metricRules[0].QueryForSeries([]string{})
	fmt.Println(expression, err)
}
