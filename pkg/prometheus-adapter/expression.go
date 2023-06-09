package prometheus_adapter

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	"sigs.k8s.io/prometheus-adapter/pkg/config"
	"sigs.k8s.io/prometheus-adapter/pkg/naming"
)

// MetricMatches for Prometheus-adapter-config
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
	MetricRulesErrorRes []string
	MetricRulesErrorCus []string
	MetricRulesErrorExt []string
	ExtensionLabels     []string
	MetricRulesError    bool
}

type MetricRule struct {
	MetricMatches string
	SeriesName    string
	ResConverter  naming.ResourceConverter
	Template      *template.Template
	Namespaced    bool
	LabelMatchers []string
}

type QueryTemplateArgs struct {
	Series        string
	LabelMatchers string
}

// get MetricRules from config.MetricsDiscoveryConfig
func GetMetricRules() *MetricRules {
	return metricRules
}

// count MetricRulesErrorRes from config.MetricsDiscoveryConfig
func CountMetricRulesErrorRes() int {
	return len(metricRules.MetricRulesErrorRes)
}

// count MetricRulesErrorCus from config.MetricsDiscoveryConfig
func CountMetricRulesErrorCus() int {
	return len(metricRules.MetricRulesErrorCus)
}

// count MetricRulesErrorExt from config.MetricsDiscoveryConfig
func CountMetricRulesErrorExt() int {
	return len(metricRules.MetricRulesErrorExt)
}

// set MetricRulesError from config.MetricsDiscoveryConfig
func SetMetricRulesError(state bool) {
	metricRules.MetricRulesError = state
}

func GetMetricRulesError() bool {
	return metricRules.MetricRulesError
}

// ParsingResourceRules from config.MetricsDiscoveryConfig
func ParsingResourceRules(mc config.MetricsDiscoveryConfig, mapper meta.RESTMapper) (err error) {
	if mc.ResourceRules == nil {
		metricRules.MetricRulesResource = []MetricRule{}
		return fmt.Errorf("ResourceRules is nil")
	}

	metricRules.MetricRulesResource, metricRules.MetricRulesErrorRes, err = GetMetricRulesFromResourceRules(*mc.ResourceRules, mapper)
	return err
}

// ParsingRules from config.MetricsDiscoveryConfig
func ParsingRules(mc config.MetricsDiscoveryConfig, mapper meta.RESTMapper) (err error) {
	if mc.Rules == nil {
		metricRules.MetricRulesCustomer = []MetricRule{}
		return fmt.Errorf("Rules is nil")
	}
	metricRules.MetricRulesCustomer, metricRules.MetricRulesErrorCus, err = GetMetricRulesFromDiscoveryRule(mc.Rules, mapper)
	return err
}

// ParsingExternalRules from config.MetricsDiscoveryConfig
func ParsingExternalRules(mc config.MetricsDiscoveryConfig, mapper meta.RESTMapper) (err error) {
	if mc.ExternalRules == nil {
		metricRules.MetricRulesExternal = []MetricRule{}
		return fmt.Errorf("ExternalRules is nil")
	}
	metricRules.MetricRulesExternal, metricRules.MetricRulesErrorExt, err = GetMetricRulesFromDiscoveryRule(mc.ExternalRules, mapper)
	return err
}

// SetExtensionLabels from opts.DataSourcePromConfig.AdapterExtensionLabels
func SetExtensionLabels(extensionLabels string) {
	if extensionLabels != "" {
		for _, label := range strings.Split(extensionLabels, ",") {
			metricRules.ExtensionLabels = append(metricRules.ExtensionLabels, label)
		}
	}
}

// GetMetricRuleResourceFromRules produces a MetricNamer for each rule in the given config.
func GetMetricRulesFromResourceRules(cfg config.ResourceRules, mapper meta.RESTMapper) (metricRules []MetricRule, metricRulesError []string, err error) {
	// get cpu MetricsQuery
	if cfg.CPU.ContainerQuery != "" {
		var resConverter naming.ResourceConverter
		var boolmetricRulePrase = true

		if cfg.CPU.Resources.Overrides != nil {
			resConverter, err = naming.NewResourceConverter(cfg.CPU.Resources.Template, cfg.CPU.Resources.Overrides, mapper)
			if err != nil {
				boolmetricRulePrase = false
				klog.Errorf("unable to construct label-resource converter: %s %v", "resource.cpu", err)
			} else {
				reg, err := regexp.Compile(`\s*by\s*\(<<.GroupBy>>\)\s*$`)
				if err != nil {
					boolmetricRulePrase = false
					klog.Errorf("unable to match <.GroupBy>")
				} else {
					queryTemplate := reg.ReplaceAllString(cfg.CPU.ContainerQuery, "")
					templ, err := template.New("metrics-query").Delims("<<", ">>").Parse(queryTemplate)
					if err != nil {
						boolmetricRulePrase = false
						klog.Errorf("unable to parse metrics query template %q: %v", cfg.CPU.ContainerQuery, err)
					} else {
						metricRules = append(metricRules, MetricRule{
							MetricMatches: "cpu",
							ResConverter:  resConverter,
							Template:      templ,
							Namespaced:    true,
						})
					}
				}
			}
		}
		if !boolmetricRulePrase {
			metricRulesError = append(metricRulesError, cfg.CPU.ContainerQuery)
		}
	}
	// get cpu MetricsQuery
	if cfg.Memory.ContainerQuery != "" {
		var resConverter naming.ResourceConverter
		var boolmetricRulePrase = true

		if cfg.Memory.Resources.Overrides != nil {
			resConverter, err = naming.NewResourceConverter(cfg.Memory.Resources.Template, cfg.Memory.Resources.Overrides, mapper)
			if err != nil {
				boolmetricRulePrase = false
				klog.Errorf("unable to construct label-resource converter: %s %v", "resource.memory", err)
			} else {
				reg, err := regexp.Compile(`\s*by\s*\(<<.GroupBy>>\)\s*$`)
				if err != nil {
					boolmetricRulePrase = false
					klog.Errorf("unable to match <.GroupBy>")
				} else {
					queryTemplate := reg.ReplaceAllString(cfg.Memory.ContainerQuery, "")
					templ, err := template.New("metrics-query").Delims("<<", ">>").Parse(queryTemplate)
					if err != nil {
						boolmetricRulePrase = false
						klog.Errorf("unable to parse metrics query template %q: %v", cfg.Memory.ContainerQuery, err)
					} else {
						metricRules = append(metricRules, MetricRule{
							MetricMatches: "memory",
							ResConverter:  resConverter,
							Template:      templ,
							Namespaced:    true,
						})
					}
				}
			}
		}
		if !boolmetricRulePrase {
			metricRulesError = append(metricRulesError, cfg.Memory.ContainerQuery)
		}
	}

	return metricRules, metricRulesError, nil
}

// GetMetricRuleFromRules produces a MetricNamer for each rule in the given config.
func GetMetricRulesFromDiscoveryRule(cfg []config.DiscoveryRule, mapper meta.RESTMapper) (metricRules []MetricRule, metricsError []string, err error) {
	for _, rule := range cfg {
		var resConverter naming.ResourceConverter
		if rule.Resources.Overrides != nil {
			resConverter, err = naming.NewResourceConverter(rule.Resources.Template, rule.Resources.Overrides, mapper)
			if err != nil {
				metricsError = append(metricsError, rule.SeriesQuery)
				klog.Errorf("unable to construct label-resource converter: %s %v", rule.SeriesQuery, err)
				continue
			}
		}

		// queries are namespaced by default unless the rule specifically disables it
		namespaced := true
		if rule.Resources.Namespaced != nil {
			namespaced = *rule.Resources.Namespaced
		}

		reg, err := regexp.Compile(`\s*by\s*\(<<.GroupBy>>\)\s*$`)
		if err != nil {
			metricsError = append(metricsError, rule.SeriesQuery)
			klog.Errorf("unable to match <.GroupBy> %s %v", rule.SeriesQuery, err)
			continue
		}
		queryTemplate := reg.ReplaceAllString(rule.MetricsQuery, "")

		templ, err := template.New("metrics-query").Delims("<<", ">>").Parse(queryTemplate)
		if err != nil {
			metricsError = append(metricsError, rule.SeriesQuery)
			klog.Errorf("unable to parse metrics query template %q: %v", queryTemplate, err)
			continue
		}

		// get seriesName from SeriesQuery
		seriesName := GetSeriesNameFromSeriesQuery(rule.SeriesQuery)

		// get labelMatchers from DiscoveryRule
		labelMatchers := GetLabelMatchersFromDiscoveryRule(rule)

		// get metricMatches from DiscoveryRule
		metricMatches, err := GetMetricMatchesFromDiscoveryRule(rule, seriesName)
		if err != nil {
			metricsError = append(metricsError, rule.SeriesQuery)
			klog.Errorf("unable to get metricMatches with DiscoveryRule %v", err)
			continue
		}
		if metricMatches == "" {
			metricsError = append(metricsError, rule.SeriesQuery)
			klog.Errorf("unable to get metricMatches with DiscoveryRule %v", rule)
			continue
		}

		metricRules = append(metricRules, MetricRule{
			MetricMatches: metricMatches,
			SeriesName:    seriesName,
			LabelMatchers: labelMatchers,
			ResConverter:  resConverter,
			Template:      templ,
			Namespaced:    namespaced,
		})
	}

	return metricRules, metricsError, nil
}

// get MetricRule for metricName
func MatchMetricRule(mrs []MetricRule, metricName string) *MetricRule {
	for _, metricRule := range mrs {
		if match, _ := (regexp.Match(metricRule.MetricMatches, []byte(metricName))); match {
			return &metricRule
		}
	}
	return nil
}

// get MetrycsQuery by naming.MetricsQuery.BuildExternal from prometheus-adapter
func (mr *MetricRule) QueryForSeries(namespace string, nameReg string, exprs []string) (expressionQuery string, err error) {
	if mr.LabelMatchers != nil {
		exprs = append(mr.LabelMatchers, exprs...)
	}

	if mr.Namespaced && namespace != "" {
		if mr.ResConverter != nil {
			namespaceLbl, err := mr.ResConverter.LabelForResource(naming.NsGroupResource)
			if err != nil {
				return "", err
			}
			exprs = append(exprs, fmt.Sprintf("%s=\"%s\"", namespaceLbl, namespace))
		} else {
			exprs = append(exprs, fmt.Sprintf("%s=\"%s\"", "namespace", namespace))
		}
	}

	if nameReg != "" {
		if mr.ResConverter != nil {
			resourceLbl, err := mr.ResConverter.LabelForResource(schema.GroupResource{Resource: "pods"})
			if err != nil {
				return "", err
			}
			exprs = append(exprs, fmt.Sprintf("%s=~\"%s\"", resourceLbl, nameReg))
		} else {
			exprs = append(exprs, fmt.Sprintf("%s=~\"%s\"", "pod", nameReg))
		}
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

// get SeriesName from SeriesQuery
func GetSeriesNameFromSeriesQuery(seriesQuery string) (seriesName string) {
	regSeriesName := regexp.MustCompile("(.*){.*}")
	if len(regSeriesName.FindStringSubmatch(seriesQuery)) > 1 {
		seriesName = regSeriesName.FindStringSubmatch(seriesQuery)[1]
	} else {
		seriesName = seriesQuery
	}
	return seriesName
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
	if len(regLabelMatchers.FindStringSubmatch(rule.SeriesQuery)) > 1 {
		SeriesMatchers := regLabelMatchers.FindStringSubmatch(rule.SeriesQuery)[1]
		if SeriesMatchers != "" {
			for _, seriesMatcher := range strings.Split(SeriesMatchers, ",") {
				labelMatchers = append(labelMatchers, seriesMatcher)
			}
		}
	}

	return labelMatchers
}

// get MetricMatches from DiscoveryRule
func GetMetricMatchesFromDiscoveryRule(rule config.DiscoveryRule, seriesName string) (metricMatches string, err error) {
	if seriesName == "" {
		regSeries := regexp.MustCompile("^{(.*)}")
		if len(regSeries.FindStringSubmatch(rule.SeriesQuery)) > 1 {
			for _, i := range strings.Split(regSeries.FindStringSubmatch(rule.SeriesQuery)[1], ",") {
				regLabelName := regexp.MustCompile("__name__[~|=]+\"(.*)\"")
				if len(regLabelName.FindStringSubmatch(i)) > 1 {
					seriesName = regLabelName.FindStringSubmatch(i)[1]
					break
				} else {
					continue
				}
			}
		}
	}

	if seriesName == "" {
		return metricMatches, fmt.Errorf("unable to get [%s] from series query %q", "__name__", rule.SeriesQuery)
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
