package prometheus_adapter

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/prometheus-adapter/pkg/config"
	"sigs.k8s.io/prometheus-adapter/pkg/naming"
)

const (
	WorkloadCpuUsageExpression  = "WorkloadCpuUsageExpression"
	WorkloadMemUsageExpression  = "WorkloadMemUsageExpression"
	NodeCpuUsageExpression      = "NodeCpuUsageExpression"
	NodeMemUsageExpression      = "NodeMemUsageExpression"
	ContainerCpuUsageExpression = "ContainerCpuUsageExpression"
	ContainerMemUsageExpression = "ContainerMemUsageExpression"
	PodCpuUsageExpression       = "PodCpuUsageExpression"
	PodMemUsageExpression       = "PodMemUsageExpression"
)

// define MetricRule for expressionQuery, SeriesName for original metric name, MetricName for name converted by prometheus-adapter
var (
	metricRules *MetricRules
)

func init() {
	metricRules = &MetricRules{}
}

type MetricRules struct {
	MetricRulesResource []MetricRule
	MetricRulesCustomer []MetricRule
	MetricRulesExternal []MetricRule
}

type MetricRule struct {
	MetricMatches string
	MetricsQuery  naming.MetricsQuery
	SeriesName    string
	Template      *template.Template
	LabelMatchers []string
}

type QueryTemplateArgs struct {
	Series        string
	LabelMatchers string
}

// GetMetricRules get metricRules from config.MetricsDiscoveryConfig
func GetMetricRules() *MetricRules {
	return metricRules
}

// FlushResourceRules from config.MetricsDiscoveryConfig
func FlushResourceRules(mc config.MetricsDiscoveryConfig, mapper apimeta.RESTMapper) (err error) {
	metricRules.MetricRulesResource, err = GetMetricRulesFromResourceRules(*mc.ResourceRules, mapper)
	return err
}

// FlushRules from config.MetricsDiscoveryConfig
func FlushRules(mc config.MetricsDiscoveryConfig, mapper apimeta.RESTMapper) (err error) {
	if mc.Rules == nil {
		return fmt.Errorf("Rules is nil")
	} else {
		metricRules.MetricRulesCustomer, err = GetMetricRulesFromDiscoveryRule(mc.Rules, mapper)
	}
	return err
}

// FlushExternalRules from config.MetricsDiscoveryConfig
func FlushExternalRules(mc config.MetricsDiscoveryConfig, mapper apimeta.RESTMapper) (err error) {
	if mc.ExternalRules == nil {
		return fmt.Errorf("ExternalRules is nil")
	} else {
		metricRules.MetricRulesExternal, err = GetMetricRulesFromDiscoveryRule(mc.ExternalRules, mapper)
	}
	return err
}

// GetMetricRuleResourceFromRules produces a MetricNamer for each rule in the given config.
func GetMetricRulesFromResourceRules(cfg config.ResourceRules, mapper apimeta.RESTMapper) ([]MetricRule, error) {
	var metricRules []MetricRule

	// get cpu MetricsQuery
	if cfg.CPU.ContainerQuery != "" {
		reg, err := regexp.Compile(` by \(.*\)$`)
		if err != nil {
			return nil, fmt.Errorf("unable to match <.GroupBy>")
		}
		queryTemplate := reg.ReplaceAllString(cfg.CPU.ContainerQuery, "")
		converter, err := naming.NewResourceConverter(cfg.CPU.Resources.Template, cfg.CPU.Resources.Overrides, mapper)
		if err != nil {
			return metricRules, fmt.Errorf("unable to construct label-resource converter: %v", err)
		}

		templ, err := template.New("metrics-query").Delims("<<", ">>").Parse(queryTemplate)
		if err != nil {
			return nil, fmt.Errorf("unable to parse metrics query template %q: %v", cfg.CPU.ContainerQuery, err)
		}

		metricQuery, err := naming.NewMetricsQuery(queryTemplate, converter)
		if err != nil {
			return metricRules, fmt.Errorf("unable to construct container metrics query: %v", err)
		}

		metricRules = append(metricRules, MetricRule{
			MetricMatches: "cpu",
			MetricsQuery:  metricQuery,
			Template:      templ,
		})
	}
	// get cpu MetricsQuery
	if cfg.Memory.ContainerQuery != "" {
		reg, err := regexp.Compile(` by \(.*\)$`)
		if err != nil {
			return nil, fmt.Errorf("unable to match <.GroupBy>")
		}
		queryTemplate := reg.ReplaceAllString(cfg.Memory.ContainerQuery, "")

		converter, err := naming.NewResourceConverter(cfg.Memory.Resources.Template, cfg.Memory.Resources.Overrides, mapper)
		if err != nil {
			return metricRules, fmt.Errorf("unable to construct label-resource converter: %v", err)
		}

		templ, err := template.New("metrics-query").Delims("<<", ">>").Parse(queryTemplate)
		if err != nil {
			return nil, fmt.Errorf("unable to parse metrics query template %q: %v", cfg.Memory.ContainerQuery, err)
		}

		metricQuery, err := naming.NewMetricsQuery(queryTemplate, converter)
		if err != nil {
			return metricRules, fmt.Errorf("unable to construct container metrics query: %v", err)
		}

		metricRules = append(metricRules, MetricRule{
			MetricMatches: "memory",
			MetricsQuery:  metricQuery,
			Template:      templ,
		})
	}

	return metricRules, nil
}

// GetMetricRuleFromRules produces a MetricNamer for each rule in the given config.
func GetMetricRulesFromDiscoveryRule(cfg []config.DiscoveryRule, mapper apimeta.RESTMapper) ([]MetricRule, error) {
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

		reg, err := regexp.Compile(` by \(.*\)$`)
		if err != nil {
			return nil, fmt.Errorf("unable to match <.GroupBy>")
		}
		queryTemplate := reg.ReplaceAllString(rule.MetricsQuery, "")

		templ, err := template.New("metrics-query").Delims("<<", ">>").Parse(queryTemplate)
		if err != nil {
			return nil, fmt.Errorf("unable to parse metrics query template %q: %v", queryTemplate, err)
		}

		metricsQuery, err := naming.NewExternalMetricsQuery(queryTemplate, resConv, namespaced)
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

		metricRules[i] = MetricRule{
			MetricMatches: metricMatches,
			MetricsQuery:  metricsQuery,
			SeriesName:    seriesName,
			LabelMatchers: labelMatchers,
			Template:      templ,
		}
	}

	return metricRules, nil
}

func MatchMetricRule(mrs []MetricRule, metricName string) *MetricRule {
	for _, metricRule := range mrs {
		if match, _ := (regexp.Match(metricRule.MetricMatches, []byte(metricName))); match {
			return &metricRule
		}
	}
	return nil
}

/*
// get MetrycsQuery by naming.MetricsQuery.Build from prometheus-adapter
func (mr *MetricRule) QueryForSeriesResource(exprs []string) (expressionQuery string, err error) {
	if mr.LabelMatchers != nil {
		exprs = append(mr.LabelMatchers, exprs...)
	}

	args := &QueryTemplateArgs{
		Series:        mr.SeriesName,
		LabelMatchers: strings.Join(exprs, ","),
	}

	queryBuff := new(bytes.Buffer)
	if err := mr.Template.Execute(queryBuff, args); err != nil {
		return "", err
	}

	if queryBuff.Len() == 0 {
		return "", fmt.Errorf("empty query produced by metrics query template")
	}

	reg, err := regexp.Compile(` by \(.*\)$`)
	if err != nil {
		return "", err
	}

	return reg.ReplaceAllString(queryBuff.String(), ""), err
	//return queryBuff.String(), err
}

// get MetrycsQuery by naming.MetricsQuery.BuildExternal from prometheus-adapter
func (mr *MetricRule) QueryForSeriesCustomer(exprs []string) (expressionQuery string, err error) {
	if mr.LabelMatchers != nil {
		exprs = append(mr.LabelMatchers, exprs...)
	}

	args := &QueryTemplateArgs{
		Series:        mr.SeriesName,
		LabelMatchers: strings.Join(exprs, ","),
	}

	queryBuff := new(bytes.Buffer)
	if err := mr.Template.Execute(queryBuff, args); err != nil {
		return "", err
	}

	if queryBuff.Len() == 0 {
		return "", fmt.Errorf("empty query produced by metrics query template")
	}

	//	reg, err := regexp.Compile(` by \(.*\)$`)
	//	if err != nil {
	//		return "", err
	//	}

	//	return reg.ReplaceAllString(queryBuff.String(), ""), err
	return queryBuff.String(), err
}

*/
// get MetrycsQuery by naming.MetricsQuery.BuildExternal from prometheus-adapter
func (mr *MetricRule) QueryForSeries(exprs []string) (expressionQuery string, err error) {
	if mr.LabelMatchers != nil {
		exprs = append(mr.LabelMatchers, exprs...)
	}

	args := &QueryTemplateArgs{
		Series:        mr.SeriesName,
		LabelMatchers: strings.Join(exprs, ","),
	}

	queryBuff := new(bytes.Buffer)
	if err := mr.Template.Execute(queryBuff, args); err != nil {
		return "", err
	}

	if queryBuff.Len() == 0 {
		return "", fmt.Errorf("empty query produced by metrics query template")
	}

	return queryBuff.String(), err
}

// get SeriesName from seriesQuery
func GetSeriesNameFromSeriesQuery(seriesQuery string) string {
	regSeriesName := regexp.MustCompile("(.*?){")
	return regSeriesName.FindStringSubmatch(seriesQuery)[1]
}

// get labelMatchers from DiscoveryRule
func GetLabelMatchersFromDiscoveryRule(rule config.DiscoveryRule) []string {
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

	return labelMatchers
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
