package utils

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

func GetGroupVersionResource(discoveryClient discovery.DiscoveryInterface, apiVersion string, kind string) (*schema.GroupVersionResource, error) {
	resList, err := discoveryClient.ServerResourcesForGroupVersion(apiVersion)
	if err != nil {
		return nil, err
	}

	var resName string
	for _, res := range resList.APIResources {
		if kind == res.Kind {
			resName = res.Name
			break
		}
	}
	if resName == "" {
		return nil, fmt.Errorf("invalid kind %s", kind)
	}

	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return nil, err
	}
	gvr := gv.WithResource(resName)
	return &gvr, nil
}
