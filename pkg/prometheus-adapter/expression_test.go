package prometheus_adapter

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/prometheus-adapter/pkg/config"
)

func TestQueryForSeriesResource(t *testing.T) {
	containerQuery := `sum(rate(container_cpu_usage_seconds_total{<<.LabelMatchers>>}[3m])) by (<<.GroupBy>>)`
	namespaced := true

	cfg := &config.ResourceRules{
		CPU: config.ResourceRule{
			ContainerQuery: containerQuery,
			Resources: config.ResourceMapping{
				Overrides: map[string]config.GroupResource{
					"pod_namespace": {Resource: "namespace"},
					"pod_name":      {Resource: "pod"},
				},
				Namespaced: &namespaced,
			},
			ContainerLabel: "container",
		},
	}

	mp := meta.NewDefaultRESTMapper([]schema.GroupVersion{corev1.SchemeGroupVersion})
	mp.Add(corev1.SchemeGroupVersion.WithKind("Pod"), meta.RESTScopeNamespace)
	mp.Add(corev1.SchemeGroupVersion.WithKind("Namespace"), meta.RESTScopeRoot)

	test := struct {
		description string
		resource    config.ResourceRules
		mapper      meta.RESTMapper
		namespace   string
		nameReg     string
		expect      string
	}{
		description: "get expressionQuery For SeriesResource",
		resource:    *cfg,
		mapper:      mp,
		namespace:   "test",
		nameReg:     "test-.*",
		expect:      "sum(rate(container_cpu_usage_seconds_total{pod_namespace=\"test\",pod_name=~\"test-.*\"}[3m]))",
	}

	metricRules, _ := GetMetricRulesFromResourceRules(test.resource, test.mapper)
	requests, err := metricRules[0].QueryForSeries(test.namespace, test.nameReg, []string{})
	if err != nil {
		t.Errorf("Failed to QueryForSeriesResource: %v", err)
	}
	if requests != test.expect {
		t.Errorf("expect requests %s actual requests %s", test.expect, requests)
	}
}

func TestQueryForSeriesRules(t *testing.T) {
	seriesQuery := `nginx_concurrent_utilization{namespace!="",pod!=""}`
	metricsQuery := `sum(<<.Series>>{<<.LabelMatchers>>}) by (<<.GroupBy>>)`
	namespaced := true

	discoveryRule := config.DiscoveryRule{
		SeriesQuery:  seriesQuery,
		MetricsQuery: metricsQuery,
		Resources: config.ResourceMapping{
			Overrides: map[string]config.GroupResource{
				"pod_namespace": {Resource: "namespace"},
				"pod_name":      {Resource: "pod"},
			},
			Namespaced: &namespaced,
		},
	}

	mp := meta.NewDefaultRESTMapper([]schema.GroupVersion{corev1.SchemeGroupVersion})
	mp.Add(corev1.SchemeGroupVersion.WithKind("Pod"), meta.RESTScopeNamespace)
	mp.Add(corev1.SchemeGroupVersion.WithKind("Namespace"), meta.RESTScopeRoot)

	test := struct {
		description string
		resource    []config.DiscoveryRule
		mapper      meta.RESTMapper
		namespace   string
		nameReg     string
		expect      string
	}{
		description: "get expressionQuery For SeriesRules",
		resource:    []config.DiscoveryRule{discoveryRule},
		mapper:      mp,
		namespace:   "test",
		nameReg:     "test-.*",
		expect:      "sum(nginx_concurrent_utilization{namespace!=\"\",pod!=\"\",pod_namespace=\"test\",pod_name=~\"test-.*\"})",
	}

	metricRules, _ := GetMetricRulesFromDiscoveryRule(test.resource, test.mapper)
	requests, err := metricRules[0].QueryForSeries(test.namespace, test.nameReg, []string{})
	if err != nil {
		t.Errorf("Failed to QueryForSeriesResource: %v", err)
	}
	if requests != test.expect {
		t.Errorf("expect requests %s actual requests %s", test.expect, requests)
	}
}

func TestGetSeriesNameFromSeriesQuery(t *testing.T) {
	seriesQuery := `nginx_concurrent_utilization{pod_namespace!="",pod_name!=""}`

	test := struct {
		description string
		resource    string
		expect      string
	}{
		description: "get SeriesName For SeriesQuery",
		resource:    seriesQuery,
		expect:      "nginx_concurrent_utilization",
	}

	requests := GetSeriesNameFromSeriesQuery(test.resource)

	if requests != test.expect {
		t.Errorf("expect requests %s actual requests %s", test.expect, requests)
	}
}

func TestGetLabelMatchersFromDiscoveryRule(t *testing.T) {
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

	test := struct {
		description string
		resource    config.DiscoveryRule
		expect      []string
	}{
		description: "get expressionQuery For SeriesRules",
		resource:    discoveryRule,
		expect:      []string{"pod_namespace!=\"\"", "pod_name!=\"\""},
	}

	requests := GetLabelMatchersFromDiscoveryRule(test.resource)
	for i := range requests {
		if requests[i] != test.expect[i] {
			t.Errorf("expect requests %s actual requests %s", test.expect, requests)
		}
	}
}

func TestGetMetricMatchesFromDiscoveryRule(t *testing.T) {
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

	test := struct {
		description string
		resource    config.DiscoveryRule
		expect      string
	}{
		description: "get expressionQuery For SeriesRules",
		resource:    discoveryRule,
		expect:      "nginx_concurrent_utilization",
	}

	requests, err := GetMetricMatchesFromDiscoveryRule(test.resource)
	if err != nil {
		t.Errorf("Failed to GetMetricMatchesFromDiscoveryRule: %v", err)
	}
	if requests != test.expect {
		t.Errorf("expect requests %s actual requests %s", test.expect, requests)
	}
}
