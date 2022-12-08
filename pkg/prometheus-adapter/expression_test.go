package prometheus_adapter

import (
	"fmt"
	"sigs.k8s.io/prometheus-adapter/pkg/config"

	"testing"
)

func TestQueryForSeriesResource(t *testing.T) {
	containerQuery := `sum(rate(container_cpu_usage_seconds_total{<<.LabelMatchers>>}[3m])) by (<<.GroupBy>>)`
	namespaced := true

	cfg := &config.ResourceRules{
		CPU: config.ResourceRule{
			ContainerQuery: containerQuery,
			Resources: config.ResourceMapping{
				Overrides:  map[string]config.GroupResource{},
				Namespaced: &namespaced,
			},
			ContainerLabel: "container",
		},
	}

	metricRules, _ := GetMetricRulesFromResourceRules(*cfg)
	expression, err := metricRules[0].QueryForSeries("test", []string{})
	fmt.Println(expression, err)
}

func TestQueryForSeriesRules(t *testing.T) {
	seriesQuery := `nginx_concurrent_utilization{pod_namespace!="",pod_name!=""}`
	metricsQuery := `sum(<<.Series>>{<<.LabelMatchers>>}) by (<<.GroupBy>>)`
	namespaced := true

	discoveryRule := config.DiscoveryRule{
		SeriesQuery:  seriesQuery,
		MetricsQuery: metricsQuery,
		Resources: config.ResourceMapping{
			Namespaced: &namespaced,
		},
	}

	metricRules, _ := GetMetricRulesFromDiscoveryRule([]config.DiscoveryRule{discoveryRule})
	expression, err := metricRules[0].QueryForSeries("test", []string{})
	fmt.Println(expression, err)
}

func TestQueryForSeriesExternalRules(t *testing.T) {
	seriesQuery := `nginx_concurrent_utilization{pod_namespace!="",pod_name!=""}`
	metricsQuery := `sum by(node, route)(rate(<<.Series>>{<<.LabelMatchers>>}[1m]))`
	namespaced := true

	discoveryRule := config.DiscoveryRule{
		SeriesQuery:  seriesQuery,
		MetricsQuery: metricsQuery,
		Resources: config.ResourceMapping{
			Namespaced: &namespaced,
		},
	}

	metricRules, _ := GetMetricRulesFromDiscoveryRule([]config.DiscoveryRule{discoveryRule})
	expression, err := metricRules[0].QueryForSeries("test", []string{})
	fmt.Println(expression, err)
}
