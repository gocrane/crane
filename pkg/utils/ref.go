package utils

import (
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	CgroupKubePods  = "kubepods"
	CgroupPodPrefix = "pod"
)

func GetNodeRef(nodeName string) *v1.ObjectReference {
	return &v1.ObjectReference{
		Kind:      "Node",
		Name:      nodeName,
		UID:       types.UID(nodeName),
		Namespace: "",
	}
}

func GetContainerIdFromKey(key string) string {
	subPaths := strings.Split(key, "/")

	if len(subPaths) > 0 {
		// if the latest sub path is pod-xxx-xxx, we regard as it od path
		// if not we used the latest sub path as the containerId
		if strings.HasPrefix(subPaths[len(subPaths)-1], CgroupPodPrefix) {
			return ""
		} else {
			return subPaths[len(subPaths)-1]
		}
	}

	return ""
}
