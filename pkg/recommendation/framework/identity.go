package framework

import unstructuredv1 "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

type ObjectIdentity struct {
	Namespace  string
	APIVersion string
	Kind       string
	Name       string
	Labels     map[string]string
	Object     unstructuredv1.Unstructured
}
