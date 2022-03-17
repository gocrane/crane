package estimator

import (
	corev1 "k8s.io/api/core/v1"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
)

type ExternalResourceEstimator struct {
	Type     string
	Priority int
}

func NewExternalResourceEstimator(spec autoscalingapi.ResourceEstimator) *ExternalResourceEstimator {
	return &ExternalResourceEstimator{
		Type:     spec.Type,
		Priority: spec.Priority,
	}
}

func (e *ExternalResourceEstimator) GetResourceEstimation(evpa *autoscalingapi.EffectiveVerticalPodAutoscaler, config map[string]string, containerName string, currRes *corev1.ResourceRequirements) (corev1.ResourceList, error) {
	for _, currentEstimator := range evpa.Status.CurrentEstimators {
		if currentEstimator.Type == e.Type {
			for _, containerRecommendation := range currentEstimator.Recommendation.ContainerRecommendations {
				if containerName == containerRecommendation.ContainerName {
					return containerRecommendation.Target, nil
				}
			}
		}
	}

	return nil, nil
}

func (e *ExternalResourceEstimator) DeleteEstimation(evpa *autoscalingapi.EffectiveVerticalPodAutoscaler) {
	// do nothing
	return
}
