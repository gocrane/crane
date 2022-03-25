package utils

import corev1 "k8s.io/api/core/v1"

func IsCPUResourceEqual(oldResource, desiredResource corev1.ResourceList) bool {
	oldCpuResource, ok1 := oldResource[corev1.ResourceCPU]
	desiredCpuResource, ok2 := desiredResource[corev1.ResourceCPU]
	if ok1 != ok2 {
		return false
	}
	if ok1 == ok2 && ok1 {
		return oldCpuResource.Equal(desiredCpuResource)
	}
	return true
}

func IsMemoryResourceEqual(oldResource, desiredResource corev1.ResourceList) bool {
	oldMemoryResource, ok1 := oldResource[corev1.ResourceMemory]
	desiredMemoryResource, ok2 := desiredResource[corev1.ResourceMemory]
	if ok1 != ok2 {
		return false
	}
	if ok1 == ok2 && ok1 {
		return oldMemoryResource.Equal(desiredMemoryResource)
	}
	return true
}

func IsResourceEqual(oldResource, desiredResource corev1.ResourceList) bool {
	// Compare CPU Resource
	if !IsCPUResourceEqual(oldResource, desiredResource) {
		return false
	}
	// Compare Memory Resource
	if !IsMemoryResourceEqual(oldResource, desiredResource) {
		return false
	}
	return true
}

func IsEqual(oldResource, desiredResource *corev1.ResourceRequirements) bool {
	// Compare Request Resource
	if !IsResourceEqual(oldResource.Requests, desiredResource.Requests) {
		return false
	}
	// Compare Limit Resource
	if !IsResourceEqual(oldResource.Limits, desiredResource.Limits) {
		return false
	}
	return true
}

func GetResourceByPodTemplate(podTemplate *corev1.PodTemplateSpec, containerName string) (*corev1.ResourceRequirements, bool) {
	for _, containerSpec := range podTemplate.Spec.Containers {
		if containerSpec.Name == containerName {
			return &containerSpec.Resources, true
		}
	}

	return nil, false
}
