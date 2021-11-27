package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func LabelSelectorMatched(maps map[string]string, selector *metav1.LabelSelector) (bool, error) {
	if selector == nil {
		return true, nil
	}

	var ls, err = metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return false, err
	}

	if ls.Matches(labels.Set(maps)) {
		return true, nil
	}

	return false, nil
}
