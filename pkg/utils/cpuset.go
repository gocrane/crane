package utils

import (
	topologyapi "github.com/gocrane/api/topology/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
)

// GetReservedCPUs ...
func GetReservedCPUs(cpus string) (cpuset.CPUSet, error) {
	emptyCPUSet := cpuset.NewCPUSet()
	if cpus == "" {
		return emptyCPUSet, nil
	}
	return cpuset.Parse(cpus)
}

// PodExcludeReservedCPUs ...
func PodExcludeReservedCPUs(pod *corev1.Pod) bool {
	if pod == nil {
		return false
	}
	return pod.Annotations[topologyapi.AnnotationPodExcludeReservedCPUs] == "true"
}
