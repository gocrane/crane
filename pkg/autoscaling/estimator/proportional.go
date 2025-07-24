package estimator

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
)

const (
	MinCpuResource    = 100
	MinMemoryResource = 256 * 1024 * 1024
)

type ProportionalResourceEstimator struct {
	client.Client
}

func (e *ProportionalResourceEstimator) GetResourceEstimation(evpa *autoscalingapi.EffectiveVerticalPodAutoscaler, config map[string]string, containerName string, currRes *corev1.ResourceRequirements) (corev1.ResourceList, error) {
	recommendResource := corev1.ResourceList{}

	cpuQuantity := currRes.Requests[corev1.ResourceCPU]
	if !cpuQuantity.IsZero() {
		cpuValue := cpuQuantity.MilliValue()
		if cpuValue > MinCpuResource {
			recommendResource[corev1.ResourceCPU] = *resource.NewMilliQuantity(cpuValue/2, resource.DecimalSI)
		}
	}

	memoryQuantity := currRes.Requests[corev1.ResourceMemory]
	if !memoryQuantity.IsZero() {
		memoryValue := memoryQuantity.Value()
		if memoryValue > MinMemoryResource {
			recommendResource[corev1.ResourceMemory] = *resource.NewQuantity(memoryValue/2, resource.BinarySI)
		}
	}

	return recommendResource, nil
}

func (e *ProportionalResourceEstimator) DeleteEstimation(evpa *autoscalingapi.EffectiveVerticalPodAutoscaler) {
	// do nothing
	return
}
