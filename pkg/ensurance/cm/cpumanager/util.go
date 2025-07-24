package cpumanager

import (
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	topologyapi "github.com/gocrane/api/topology/v1alpha1"
)

var (
	// SupportedPolicy is the valid cpu policy.
	SupportedPolicy = sets.NewString(
		topologyapi.AnnotationPodCPUPolicyNone, topologyapi.AnnotationPodCPUPolicyExclusive,
		topologyapi.AnnotationPodCPUPolicyNUMA, topologyapi.AnnotationPodCPUPolicyImmovable,
	)
)

// GetPodTopologyResult returns the Topology scheduling result of a pod.
func GetPodTopologyResult(pod *corev1.Pod) topologyapi.ZoneList {
	raw, exist := pod.Annotations[topologyapi.AnnotationPodTopologyResultKey]
	if !exist {
		return nil
	}
	var zones topologyapi.ZoneList
	if err := json.Unmarshal([]byte(raw), &zones); err != nil {
		return nil
	}
	return zones
}

// GetPodNUMANodeResult returns the NUMA node scheduling result of a pod.
func GetPodNUMANodeResult(pod *corev1.Pod) topologyapi.ZoneList {
	zones := GetPodTopologyResult(pod)
	var numaZones topologyapi.ZoneList
	for i := range zones {
		if zones[i].Type == topologyapi.ZoneTypeNode {
			numaZones = append(numaZones, zones[i])
		}
	}
	return numaZones
}

// GetPodTargetContainerIndices returns all pod whose cpus could be allocated.
func GetPodTargetContainerIndices(pod *corev1.Pod) []int {
	if policy := GetPodCPUPolicy(pod.Annotations); policy == topologyapi.AnnotationPodCPUPolicyNone {
		return nil
	}
	var idx []int
	for i := range pod.Spec.Containers {
		if GuaranteedCPUs(&pod.Spec.Containers[i]) > 0 {
			idx = append(idx, i)
		}
	}
	return idx
}

// GetPodCPUPolicy returns the cpu policy of pod, only supports none, exclusive, numa and immovable.
func GetPodCPUPolicy(attr map[string]string) string {
	policy, ok := attr[topologyapi.AnnotationPodCPUPolicyKey]
	if ok && SupportedPolicy.Has(policy) {
		return policy
	}
	return ""
}

// GuaranteedCPUs returns CPUs for guaranteed container.
func GuaranteedCPUs(container *corev1.Container) int {
	cpuQuantity := container.Resources.Requests[corev1.ResourceCPU]
	cpuQuantityLimit := container.Resources.Limits[corev1.ResourceCPU]

	// If requests.cpu != limits.cpu or cpu is not an integer, there are no guaranteed cpus.
	if cpuQuantity.Cmp(cpuQuantityLimit) != 0 || cpuQuantity.Value()*1000 != cpuQuantity.MilliValue() {
		return 0
	}
	// Safe downcast to do for all systems with < 2.1 billion CPUs.
	// Per the language spec, `int` is guaranteed to be at least 32 bits wide.
	// https://golang.org/ref/spec#Numeric_types
	return int(cpuQuantity.Value())
}
