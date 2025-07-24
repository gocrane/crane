package utils

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func KindForResource(resource string, restMapper meta.RESTMapper) (string, error) {
	singular, err := restMapper.ResourceSingularizer(resource)
	if err != nil {
		return "", err
	}
	fullySpecifiedGVR, groupResource := schema.ParseResourceArg(singular)
	gvk := schema.GroupVersionKind{}
	if fullySpecifiedGVR != nil {
		gvk, err = restMapper.KindFor(*fullySpecifiedGVR)
	}
	if gvk.Empty() {
		gvk, err = restMapper.KindFor(groupResource.WithVersion(""))
	}

	return gvk.Kind, err
}
