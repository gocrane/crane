package config

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gocrane/crane/pkg/utils/log"

	"github.com/gocrane/api/prediction/v1alpha1"
)

var UpdateEventBroadcaster Broadcaster = NewBroadcaster()
var DeleteEventBroadcaster Broadcaster = NewBroadcaster()

var logger = log.Logger()

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

func DeleteConfig(conf Config) {
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
