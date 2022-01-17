package analytics

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMatch(t *testing.T) {
	matchLabels := make(map[string]string)
	matchLabels["key1"] = "value1"

	unmatchLabels := make(map[string]string)
	unmatchLabels["key2"] = "value2"

	tests := []struct {
		description   string
		matchLabels   map[string]string
		labelSelector metav1.LabelSelector
		expect        bool
	}{
		{
			description: "match labels",
			matchLabels: matchLabels,
			labelSelector: metav1.LabelSelector{
				MatchLabels:      unmatchLabels,
				MatchExpressions: []metav1.LabelSelectorRequirement{},
			},
			expect: false,
		},
		{
			description: "match expression key2 exists",
			matchLabels: matchLabels,
			labelSelector: metav1.LabelSelector{
				MatchLabels: matchLabels,
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "key2",
					Operator: "Exists",
					Values:   []string{},
				}},
			},
			expect: false,
		},
		{
			description: "match expression key1 exists",
			matchLabels: matchLabels,
			labelSelector: metav1.LabelSelector{
				MatchLabels: matchLabels,
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "key1",
					Operator: "Exists",
					Values:   []string{},
				}},
			},
			expect: true,
		},
		{
			description: "match expression key1 doesNotExists",
			matchLabels: matchLabels,
			labelSelector: metav1.LabelSelector{
				MatchLabels: matchLabels,
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "key1",
					Operator: "DoesNotExist",
					Values:   []string{},
				}},
			},
			expect: false,
		},
		{
			description: "match expression key2 doesNotExists",
			matchLabels: matchLabels,
			labelSelector: metav1.LabelSelector{
				MatchLabels: matchLabels,
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "key2",
					Operator: "DoesNotExist",
					Values:   []string{},
				}},
			},
			expect: true,
		},
		{
			description: "match expression key2 in",
			matchLabels: matchLabels,
			labelSelector: metav1.LabelSelector{
				MatchLabels: matchLabels,
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "key2",
					Operator: "In",
					Values:   []string{},
				}},
			},
			expect: false,
		},
		{
			description: "match expression key1 in value1",
			matchLabels: matchLabels,
			labelSelector: metav1.LabelSelector{
				MatchLabels: matchLabels,
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "key1",
					Operator: "In",
					Values:   []string{"value1"},
				}},
			},
			expect: true,
		},
		{
			description: "match expression key1 in value2",
			matchLabels: matchLabels,
			labelSelector: metav1.LabelSelector{
				MatchLabels: matchLabels,
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "key1",
					Operator: "In",
					Values:   []string{"value2"},
				}},
			},
			expect: false,
		},
		{
			description: "match expression key1 notIn value1",
			matchLabels: matchLabels,
			labelSelector: metav1.LabelSelector{
				MatchLabels: matchLabels,
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "key1",
					Operator: "NotIn",
					Values:   []string{"value1"},
				}},
			},
			expect: false,
		},
		{
			description: "match expression key1 notIn Value2",
			matchLabels: matchLabels,
			labelSelector: metav1.LabelSelector{
				MatchLabels: matchLabels,
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "key1",
					Operator: "NotIn",
					Values:   []string{"value2"},
				}},
			},
			expect: true,
		},
	}

	for _, test := range tests {
		isMatched := match(test.labelSelector, test.matchLabels)

		if isMatched != test.expect {
			t.Errorf("%s: expect match %v actual match %v", test.description, test.expect, isMatched)
		}
	}
}
