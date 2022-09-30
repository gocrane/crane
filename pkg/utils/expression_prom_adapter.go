package utils

import (
	"fmt"
	"regexp"
	"strings"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/prometheus-adapter/pkg/config"
	"sigs.k8s.io/prometheus-adapter/pkg/naming"
)

// define MetricRule for expressionQuery, SeriesName for original metric name, MetricName for name converted by prometheus-adapter
type MetricRule struct {
	MetricMatches string
	MetricsQuery  naming.MetricsQuery
	SeriesName    string
	LabelMatchers string
}

// GetMetricRules get metricRules from config.MetricsDiscoveryConfig
func GetMetricRules(mc config.MetricsDiscoveryConfig, mapper apimeta.RESTMapper) (metricResource []MetricRule, metricCustomer []MetricRule, metricExternel []MetricRule, err error) {
	if mc.ResourceRules != nil {
		metricResource, err = GetMetricRuleResourceFromRules(*mc.ResourceRules, mapper)
		if err != nil {
			return metricResource, metricCustomer, metricExternel, err
		}
	}

	if mc.Rules != nil {
		metricCustomer, err = GetMetricRuleFromRules(mc.Rules, mapper)
		if err != nil {
			return metricResource, metricCustomer, metricExternel, err
		}
	}

	if mc.ExternalRules != nil {
		metricExternel, err = GetMetricRuleFromRules(mc.ExternalRules, mapper)
		if err != nil {
			return metricResource, metricCustomer, metricExternel, err
		}
	}
	return metricResource, metricCustomer, metricExternel, err
}

// GetMetricRuleResourceFromRules produces a MetricNamer for each rule in the given config.
func GetMetricRuleResourceFromRules(cfg config.ResourceRules, mapper apimeta.RESTMapper) ([]MetricRule, error) {
	var metricRules []MetricRule
	var metricRule MetricRule

	// get cpu MetricsQuery
	if cfg.CPU.ContainerQuery != "" {
		converter, err := naming.NewResourceConverter(cfg.CPU.Resources.Template, cfg.CPU.Resources.Overrides, mapper)
		if err != nil {
			return metricRules, fmt.Errorf("unable to construct label-resource converter: %v", err)
		}

		metricQuery, err := naming.NewMetricsQuery(cfg.CPU.ContainerQuery, converter)
		if err != nil {
			return metricRules, fmt.Errorf("unable to construct container metrics query: %v", err)
		}

		metricRule = MetricRule{
			MetricMatches: "cpu",
			MetricsQuery:  metricQuery,
		}
		metricRules = append(metricRules, metricRule)
	}
	// get cpu MetricsQuery
	if cfg.Memory.ContainerQuery != "" {
		converter, err := naming.NewResourceConverter(cfg.Memory.Resources.Template, cfg.Memory.Resources.Overrides, mapper)
		if err != nil {
			return metricRules, fmt.Errorf("unable to construct label-resource converter: %v", err)
		}

		metricQuery, err := naming.NewMetricsQuery(cfg.Memory.ContainerQuery, converter)
		if err != nil {
			return metricRules, fmt.Errorf("unable to construct container metrics query: %v", err)
		}

		metricRule = MetricRule{
			MetricMatches: "memory",
			MetricsQuery:  metricQuery,
		}
		metricRules = append(metricRules, metricRule)
	}

	return metricRules, nil
}

// GetMetricRuleFromRules produces a MetricNamer for each rule in the given config.
func GetMetricRuleFromRules(cfg []config.DiscoveryRule, mapper apimeta.RESTMapper) ([]MetricRule, error) {
	metricRules := make([]MetricRule, len(cfg))

	for i, rule := range cfg {
		resConv, err := naming.NewResourceConverter(rule.Resources.Template, rule.Resources.Overrides, mapper)
		if err != nil {
			return nil, err
		}

		// queries are namespaced by default unless the rule specifically disables it
		namespaced := true
		if rule.Resources.Namespaced != nil {
			namespaced = *rule.Resources.Namespaced
		}

		metricsQuery, err := naming.NewExternalMetricsQuery(rule.MetricsQuery, resConv, namespaced)
		if err != nil {
			return nil, fmt.Errorf("unable to construct metrics query associated with series query %q: %v", rule.SeriesQuery, err)
		}

		// get seriesName from SeriesQuery
		seriesName := GetSeriesNameFromSeriesQuery(rule.SeriesQuery)

		// get labelMatchers from DiscoveryRule
		labelMatchers := GetLabelMatchersFromDiscoveryRule(rule)

		// get metricMatches from DiscoveryRule
		metricMatches, err := GetMetricMatchesFromDiscoveryRule(rule)
		if err != nil {
			return metricRules, err
		}
		if metricMatches == "" {
			return metricRules, fmt.Errorf("unable to get metricMatches with DiscoveryRule %v", rule)
		}

		metricRule := MetricRule{
			MetricMatches: metricMatches,
			MetricsQuery:  metricsQuery,
			SeriesName:    seriesName,
			LabelMatchers: labelMatchers,
		}

		metricRules[i] = metricRule
	}

	return metricRules, nil
}

// get MetrycsQuery by naming.MetricsQuery.Build from prometheus-adapter
func (mr *MetricRule) QueryForSeriesResource(namespace string, metricSelector labels.Selector, names ...string) (expressionQuery string, err error) {
	selector, err := mr.MetricsQuery.Build("", schema.GroupResource{Resource: "pods"}, namespace, nil, metricSelector, names...)
	if err != nil {
		return "", err
	}

	reg, err := regexp.Compile(` by \(.*\)$`)
	if err != nil {
		return "", err
	}

	return reg.ReplaceAllString(string(selector), ""), err
}

// get MetrycsQuery by naming.MetricsQuery.BuildExternal from prometheus-adapter
func (mr *MetricRule) QueryForSeriesCustomer(series string, namespace string, metricSelector labels.Selector) (expressionQuery string, err error) {
	selector, err := mr.MetricsQuery.BuildExternal(series, namespace, "", []string{}, metricSelector)
	if err != nil {
		return "", err
	}

	expressionQuery = string(selector)
	if mr.LabelMatchers != "" {
		if strings.Contains(expressionQuery, "{}") {
			expressionQuery = strings.Replace(expressionQuery, "{", fmt.Sprintf("{%s", mr.LabelMatchers), 1)
		} else {
			expressionQuery = strings.Replace(expressionQuery, "{", fmt.Sprintf("{%s,", mr.LabelMatchers), 1)
		}
	}

	reg, err := regexp.Compile(` by \(.*\)$`)
	if err != nil {
		return "", err
	}

	return reg.ReplaceAllString(expressionQuery, ""), err
}

// get MetrycsQuery by naming.MetricsQuery.BuildExternal from prometheus-adapter
func (mr *MetricRule) QueryForSeriesExternal(series string, namespace string, metricSelector labels.Selector) (expressionQuery string, err error) {
	selector, err := mr.MetricsQuery.BuildExternal(series, namespace, "", []string{}, metricSelector)
	if err != nil {
		return "", err
	}

	expressionQuery = string(selector)
	if mr.LabelMatchers != "" {
		if strings.Contains(expressionQuery, "{}") {
			expressionQuery = strings.Replace(expressionQuery, "{", fmt.Sprintf("{%s", mr.LabelMatchers), 1)
		} else {
			expressionQuery = strings.Replace(expressionQuery, "{", fmt.Sprintf("{%s,", mr.LabelMatchers), 1)
		}
	}

	return expressionQuery, err
}

// get SeriesName from seriesQuery
func GetSeriesNameFromSeriesQuery(seriesQuery string) string {
	regSeriesName := regexp.MustCompile("(.*?){")
	return regSeriesName.FindStringSubmatch(seriesQuery)[1]
}

// get labelMatchers from DiscoveryRule
func GetLabelMatchersFromDiscoveryRule(rule config.DiscoveryRule) string {
	var labelMatchers []string
	if GetSeriesNameFromSeriesQuery(rule.SeriesQuery) == "" {
		// add Name Matches
		if rule.Name.Matches != "" {
			labelMatchers = append(labelMatchers, fmt.Sprintf("__name__=~\"%s\"", rule.Name.Matches))
		}
		// add SeriesFilters
		if len(rule.SeriesFilters) > 0 {
			for _, f := range rule.SeriesFilters {
				if f.Is != "" {
					labelMatchers = append(labelMatchers, fmt.Sprintf("__name__=~\"%s\"", f.Is))
				}
				if f.IsNot != "" {
					labelMatchers = append(labelMatchers, fmt.Sprintf("__name__!~\"%s\"", f.IsNot))
				}
			}
		}
	}

	// add SeriesQueryLabels
	regLabelMatchers := regexp.MustCompile("{(.*?)}")
	SeriesMatchers := regLabelMatchers.FindStringSubmatch(rule.SeriesQuery)[1]
	if SeriesMatchers != "" {
		labelMatchers = append(labelMatchers, SeriesMatchers)
	}

	return strings.Join(labelMatchers, ",")
}

// get MetricMatches from DiscoveryRule
func GetMetricMatchesFromDiscoveryRule(rule config.DiscoveryRule) (metricMatches string, err error) {
	seriesName := GetSeriesNameFromSeriesQuery(rule.SeriesQuery)
	if seriesName == "" {
		regLabelName := regexp.MustCompile("__name__[~|=]+\"(.*?)\"")
		if len(regLabelName.FindStringSubmatch(rule.SeriesQuery)) > 1 {
			seriesName = regLabelName.FindStringSubmatch(rule.SeriesQuery)[1]
		} else {
			return metricMatches, fmt.Errorf("unable to get [%s] from series query %q", "__name__", rule.SeriesQuery)
		}
	}

	var nameMatches *regexp.Regexp
	if rule.Name.Matches != "" {
		nameMatches, err = regexp.Compile(rule.Name.Matches)
		if err != nil {
			return metricMatches, fmt.Errorf("unable to compile series name match expression %q associated with series query %q: %v", rule.Name.Matches, rule.SeriesQuery, err)
		}
	} else {
		// this will always succeed
		nameMatches = regexp.MustCompile(".*")
	}
	nameAs := rule.Name.As
	if nameAs == "" {
		// check if we have an obvious default
		subexpNames := nameMatches.SubexpNames()
		if len(subexpNames) == 1 {
			// no capture groups, use the whole thing
			nameAs = "$0"
		} else if len(subexpNames) == 2 {
			// one capture group, use that
			nameAs = "$1"
		} else {
			return metricMatches, fmt.Errorf("must specify an 'as' value for name matcher %q associated with series query %q", rule.Name.Matches, rule.SeriesQuery)
		}
	}
	// get MetricName
	matches := nameMatches.FindStringSubmatchIndex(seriesName)
	if matches == nil {
		return metricMatches, fmt.Errorf("series name %q did not match expected pattern %q", seriesName, nameMatches.String())
	}
	outNameBytes := nameMatches.ExpandString(nil, nameAs, seriesName, matches)

	return string(outNameBytes), err
}
