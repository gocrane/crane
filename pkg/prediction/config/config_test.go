package config

import (
	"testing"

	"github.com/gocrane/api/prediction/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestMetricSelector_String(t *testing.T) {
	m := &v1alpha1.MetricQuery{
		MetricName: "xyz",
		QueryConditions: []v1alpha1.QueryCondition{
			{Key: "c", Operator: v1alpha1.OperatorEqual, Value: []string{"3", "2", "1"}},
			{Key: "b", Operator: v1alpha1.OperatorNotEqual, Value: []string{"a"}},
			{Key: "a", Operator: v1alpha1.OperatorRegexMatch, Value: []string{"1.5"}},
		},
	}
	assert.Equal(t, "xyz{a=~[1.5],b!=[a],c=[1,2,3]}", metricSelectorToQueryExpr(m))
}
