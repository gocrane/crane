package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func LabelSelectorMatched(maps map[string]string, selector *metav1.LabelSelector) (bool, error) {
	if selector == nil {
		return true, nil
	}

	ls, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return false, err
	}

	return ls.Matches(labels.Set(maps)), nil
}
