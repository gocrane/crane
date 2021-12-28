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

//ContainMaps to judge the maps b is contained by maps a
func ContainMaps(a map[string]string, b map[string]string) bool {
	for k, v := range b {
		if vv, ok := a[k]; !ok {
			return false
		} else {
			if vv != v {
				return false
			}
		}
	}
	return true
}
