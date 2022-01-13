package analytics

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMatch(t *testing.T) {

	testLabelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app": "nginx",
		},
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      "kubernetes.io/os",
				Operator: metav1.LabelSelectorOpExists,
			},
			{
				Key:      "kubernetes.io/gpu",
				Operator: metav1.LabelSelectorOpDoesNotExist,
			},
			{
				Key:      "type",
				Operator: metav1.LabelSelectorOpIn,
				Values:   []string{"ssd", "gpu"},
			},
			{
				Key:      "not-in-test",
				Operator: metav1.LabelSelectorOpNotIn,
				Values:   []string{"not-in-a", "not-in-b"},
			},
		},
	}

	testMatchLabels := map[string]string{
		"app": "caddy",
	}

	if match(testLabelSelector, testMatchLabels) != false {
		t.Errorf("expect false, but true")
	}

	testMatchLabels["app"] = "nginx"
	if match(testLabelSelector, testMatchLabels) != false {
		t.Errorf("expect false, but true")
	}

	testLabelSelector.MatchExpressions = testLabelSelector.MatchExpressions[1:]
	testMatchLabels["kubernetes.io/gpu"] = ""
	if match(testLabelSelector, testMatchLabels) != false {
		t.Errorf("expect false, but true")
	}

	delete(testMatchLabels, "kubernetes.io/gpu")
	if match(testLabelSelector, testMatchLabels) != false {
		t.Errorf("expect false, but true")
	}

	testMatchLabels["type"] = "cpu"
	if match(testLabelSelector, testMatchLabels) != false {
		t.Errorf("expect false, but true")
	}
	testMatchLabels["type"] = "gpu"

	testMatchLabels["not-in-test"] = "not-in-b"
	if match(testLabelSelector, testMatchLabels) != false {
		t.Errorf("expect false, but true")
	}

	testMatchLabels["not-in-test"] = ""
	if match(testLabelSelector, testMatchLabels) != true {
		t.Errorf("expect true, but false")
	}

	delete(testMatchLabels, "not-in-test")
	if match(testLabelSelector, testMatchLabels) != true {
		t.Errorf("expect true, but false")
	}

}
