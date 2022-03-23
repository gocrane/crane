package utils

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
)

func GetEVPAFromScaleTarget(context context.Context, kubeClient client.Client, namespace string, objRef corev1.ObjectReference) (*autoscalingapi.EffectiveVerticalPodAutoscaler, error) {
	evpaList := &autoscalingapi.EffectiveVerticalPodAutoscalerList{}
	opts := []client.ListOption{
		client.InNamespace(namespace),
	}
	err := kubeClient.List(context, evpaList, opts...)
	if err != nil {
		return nil, err
	}

	for _, evpa := range evpaList.Items {
		if evpa.Spec.TargetRef.Name == objRef.Name &&
			evpa.Spec.TargetRef.Kind == objRef.APIVersion &&
			evpa.Spec.TargetRef.APIVersion == objRef.APIVersion {
			return &evpa, nil
		}
	}

	return nil, nil
}
