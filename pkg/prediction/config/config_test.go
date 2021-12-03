package config

import (
	"testing"

	"github.com/gocrane/api/prediction/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestMetricSelector_String(t *testing.T) {
	m := &v1alpha1.MetricSelector{
		MetricName: "xyz",
		QueryConditions: []v1alpha1.QueryCondition{
			{"c", v1alpha1.OperatorEqual, []string{"3", "2", "1"}},
			{"b", v1alpha1.OperatorNotEqual, []string{"a"}},
			{"a", v1alpha1.OperatorRegexMatch, []string{"1.5"}},
		},
	}
	assert.Equal(t, "xyz{a=~[1.5],b!=[a],c=[1,2,3]}", metricSelectorToQueryExpr(m))
}
