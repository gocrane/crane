package analyzer

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	unstructuredv1 "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
)

type ObjectIdentity struct {
	Namespace  string
	APIVersion string
	Kind       string
	Name       string
	Labels     map[string]string
}

func getIdentities(discoveryClient discovery.DiscoveryInterface, dynamicClient dynamic.Interface, ResourceSelectors []ensuranceapi.ResourceSelector) (map[string]ObjectIdentity, error) {
	identities := map[string]ObjectIdentity{}

	for _, rs := range ResourceSelectors {
		if rs.Kind == "" {
			return nil, fmt.Errorf("empty kind")
		}

		resList, err := discoveryClient.ServerResourcesForGroupVersion(rs.APIVersion)
		if err != nil {
			return nil, err
		}

		var resName string
		for _, res := range resList.APIResources {
			klog.V(6).Infof("res %s, %s", res.Kind, res.Name)
			if rs.Kind == res.Kind {
				resName = res.Name
				break
			}
		}
		if resName == "" {
			return nil, fmt.Errorf("invalid kind %s", rs.Kind)
		}

		gv, err := schema.ParseGroupVersion(rs.APIVersion)
		if err != nil {
			return nil, err
		}
		gvr := gv.WithResource(resName)

		var unstructureds []unstructuredv1.Unstructured

		if rs.Name != "" {
			unstructured, err := dynamicClient.Resource(gvr).Namespace(rs.NameSpace).Get(context.Background(), rs.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			klog.V(6).Infof("unstructureds %s", unstructured.GetName())
			unstructureds = append(unstructureds, *unstructured)
		} else {
			var unstructuredList *unstructuredv1.UnstructuredList
			var err error

			if rs.NameSpace == "" {
				unstructuredList, err = dynamicClient.Resource(gvr).List(context.Background(), metav1.ListOptions{})
			} else {
				unstructuredList, err = dynamicClient.Resource(gvr).Namespace(rs.NameSpace).List(context.Background(), metav1.ListOptions{})
			}
			if err != nil {
				return nil, err
			}

			for _, item := range unstructuredList.Items {
				// todo: rename rs.LabelSelector to rs.matchLabelSelector ?
				m, ok, err := unstructuredv1.NestedStringMap(item.Object, "spec", "selector", "matchLabels")
				if !ok || err != nil {
					return nil, fmt.Errorf("%s not supported", gvr.String())
				}
				matchLabels := map[string]string{}
				for k, v := range m {
					matchLabels[k] = v
				}
				if labelMatch(*rs.LabelSelector, matchLabels) {
					unstructureds = append(unstructureds, item)
				}
			}
		}

		for i := range unstructureds {
			k := objRefKey(rs.Kind, rs.APIVersion, unstructureds[i].GetNamespace(), unstructureds[i].GetName())
			if _, exists := identities[k]; !exists {
				identities[k] = ObjectIdentity{
					Namespace:  unstructureds[i].GetNamespace(),
					Name:       unstructureds[i].GetName(),
					Kind:       rs.Kind,
					APIVersion: rs.APIVersion,
					Labels:     unstructureds[i].GetLabels(),
				}
			}
		}
	}

	return identities, nil
}

func objRefKey(kind, apiVersion, namespace, name string) string {
	return fmt.Sprintf("%s#%s#%s#%s", kind, apiVersion, namespace, name)
}

func labelMatch(labelSelector metav1.LabelSelector, matchLabels map[string]string) bool {
	for k, v := range labelSelector.MatchLabels {
		if matchLabels[k] != v {
			return false
		}
	}

	for _, expr := range labelSelector.MatchExpressions {
		switch expr.Operator {
		case metav1.LabelSelectorOpExists:
			if _, exists := matchLabels[expr.Key]; !exists {
				return false
			}
		case metav1.LabelSelectorOpDoesNotExist:
			if _, exists := matchLabels[expr.Key]; exists {
				return false
			}
		case metav1.LabelSelectorOpIn:
			if v, exists := matchLabels[expr.Key]; !exists {
				return false
			} else {
				var found bool
				for i := range expr.Values {
					if expr.Values[i] == v {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
		case metav1.LabelSelectorOpNotIn:
			if v, exists := matchLabels[expr.Key]; exists {
				for i := range expr.Values {
					if expr.Values[i] == v {
						return false
					}
				}
			}
		}
	}

	return true
}

func match(pod *v1.Pod, podQOS *ensuranceapi.PodQOS) bool {

	if podQOS.Spec.ScopeSelector == nil &&
		podQOS.Spec.LabelSelector.MatchLabels == nil &&
		podQOS.Spec.LabelSelector.MatchExpressions == nil {
		return false
	}

	// AND of the selectors
	var nameSpaceSelectors, prioritySelectors, qosClassSelectors []ensuranceapi.ScopedResourceSelectorRequirement
	for _, ss := range podQOS.Spec.ScopeSelector.MatchExpressions {
		if ss.ScopeName == ensuranceapi.NamespaceSelectors {
			nameSpaceSelectors = append(nameSpaceSelectors, ss)
		}
		if ss.ScopeName == ensuranceapi.PrioritySelectors {
			prioritySelectors = append(prioritySelectors, ss)
		}
		if ss.ScopeName == ensuranceapi.QOSClassSelector {
			qosClassSelectors = append(qosClassSelectors, ss)
		}
	}

	// namespace selector must be satisfied
	for _, nss := range nameSpaceSelectors {
		match, err := podMatchesNameSpaceSelector(pod, nss)
		if err != nil {
			klog.Errorf("Error on matching scope %s: %v", podQOS.Name, err)
			return false
		}
		if !match {
			klog.V(6).Infof("SvcQOS %s namespace selector not match pod %s/%s", podQOS.Name, pod.Namespace, pod.Name)
			return false
		}
	}

	var priorityTotalMatch = true
	for _, selector := range prioritySelectors {
		var priorityMatch bool
		switch selector.Operator {
		case v1.ScopeSelectorOpIn:
			for _, vaules := range selector.Values {
				priority := strings.Split(vaules, "-")
				// In format of 1000
				if len(priority) == 1 {
					p, err := strconv.Atoi(priority[0])
					if err == nil && int(*pod.Spec.Priority) == p {
						priorityMatch = true
					}
					if err != nil {
						klog.Errorf("%s can't transfer to int", priority[0])
					}
				}
				//In format of 1000-3000
				if len(priority) == 2 {
					priStart, err1 := strconv.Atoi(priority[0])
					priEnd, err2 := strconv.Atoi(priority[1])
					if err1 == nil && err2 == nil && priEnd >= priStart && (int(*pod.Spec.Priority) <= priEnd) && (int(*pod.Spec.Priority) >= priStart) {
						priorityMatch = true
					}
				}
			}
		case v1.ScopeSelectorOpNotIn:
			for _, vaules := range selector.Values {
				priority := strings.Split(vaules, "-")
				// In format of 1000
				priorityMatch = true
				if len(priority) == 1 {
					p, err := strconv.Atoi(priority[0])
					if err == nil && int(*pod.Spec.Priority) == p {
						priorityMatch = false
					}
					if err != nil {
						klog.Errorf("%s can't transfer to int", priority[0])
					}
				}
				//In format of 1000-3000
				if len(priority) == 2 {
					priStart, err1 := strconv.Atoi(priority[0])
					priEnd, err2 := strconv.Atoi(priority[1])
					if err1 == nil && err2 == nil && priEnd >= priStart && (int(*pod.Spec.Priority) <= priEnd) && (int(*pod.Spec.Priority) >= priStart) {
						priorityMatch = false
					}
				}
			}
		}
		priorityTotalMatch = priorityTotalMatch && priorityMatch
		if priorityMatch == false {
			break
		}
	}
	if !priorityTotalMatch {
		return false
	}

	var qosClassMatch = true
	for _, qos := range qosClassSelectors {
		match, err := podMatchesqosClassSelector(pod, qos)
		if err != nil {
			klog.Errorf("Error on matching scope %s: %v", podQOS.Name, err)
			qosClassMatch = false
		}
		if !match {
			klog.V(6).Infof("SvcQOS %s qosclass selector not match pod %s/%s", podQOS.Name, pod.Namespace, pod.Name)
			qosClassMatch = false
		}
	}
	if !qosClassMatch {
		return false
	}

	return true
}
func podMatchesNameSpaceSelector(pod *v1.Pod, selector ensuranceapi.ScopedResourceSelectorRequirement) (bool, error) {
	labelSelector, err := scopedResourceSelectorRequirementsAsSelector(selector)
	if err != nil {
		return false, fmt.Errorf("failed to parse and convert selector: %v", err)
	}
	m := map[string]string{string(selector.ScopeName): pod.Namespace}
	if labelSelector.Matches(labels.Set(m)) {
		return true, nil
	}
	return false, nil
}

// scopedResourceSelectorRequirementsAsSelector converts the ScopedResourceSelectorRequirement api type into a struct that implements
// labels.Selector.
func scopedResourceSelectorRequirementsAsSelector(nss ensuranceapi.ScopedResourceSelectorRequirement) (labels.Selector, error) {
	selector := labels.NewSelector()
	var op selection.Operator
	switch nss.Operator {
	case v1.ScopeSelectorOpIn:
		op = selection.In
	case v1.ScopeSelectorOpNotIn:
		op = selection.NotIn
	default:
		return nil, fmt.Errorf("%q is not a valid scope selector operator", nss.Operator)
	}
	r, err := labels.NewRequirement(string(nss.ScopeName), op, nss.Values)
	if err != nil {
		return nil, err
	}
	selector = selector.Add(*r)
	return selector, nil
}

func podMatchesqosClassSelector(pod *v1.Pod, selector ensuranceapi.ScopedResourceSelectorRequirement) (bool, error) {
	labelSelector, err := scopedResourceSelectorRequirementsAsSelector(selector)
	if err != nil {
		return false, fmt.Errorf("failed to parse and convert selector: %v", err)
	}
	m := map[string]string{string(selector.ScopeName): string(pod.Status.QOSClass)}
	if labelSelector.Matches(labels.Set(m)) {
		return true, nil
	}
	return false, nil
}
