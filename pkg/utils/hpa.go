package utils

import (
	"context"
	"fmt"

	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	"github.com/gocrane/crane/pkg/known"
)

func GetHPAFromScaleTarget(context context.Context, kubeClient client.Client, namespace string, objRef corev1.ObjectReference) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	hpaList := &autoscalingv2.HorizontalPodAutoscalerList{}
	opts := []client.ListOption{
		client.InNamespace(namespace),
	}
	err := kubeClient.List(context, hpaList, opts...)
	if err != nil {
		return nil, err
	}

	for _, hpa := range hpaList.Items {
		// bypass hpa that controller by ehpa
		if hpa.Labels != nil && hpa.Labels["app.kubernetes.io/managed-by"] == known.EffectiveHorizontalPodAutoscalerManagedBy {
			continue
		}

		if hpa.Spec.ScaleTargetRef.Name == objRef.Name &&
			hpa.Spec.ScaleTargetRef.Kind == objRef.Kind &&
			hpa.Spec.ScaleTargetRef.APIVersion == objRef.APIVersion {
			return &hpa, nil
		}
	}

	return nil, fmt.Errorf("HPA not found")
}

func GetEHPAFromScaleTarget(context context.Context, kubeClient client.Client, namespace string, objRef corev1.ObjectReference) (*autoscalingapi.EffectiveHorizontalPodAutoscaler, error) {
	ehpaList := &autoscalingapi.EffectiveHorizontalPodAutoscalerList{}
	opts := []client.ListOption{
		client.InNamespace(namespace),
	}
	err := kubeClient.List(context, ehpaList, opts...)
	if err != nil {
		return nil, err
	}

	for _, ehpa := range ehpaList.Items {
		if ehpa.Spec.ScaleTargetRef.Name == objRef.Name &&
			ehpa.Spec.ScaleTargetRef.Kind == objRef.Kind &&
			ehpa.Spec.ScaleTargetRef.APIVersion == objRef.APIVersion {
			return &ehpa, nil
		}
	}

	return nil, nil
}

func IsHPAControlledByEHPA(hpa *autoscalingv2.HorizontalPodAutoscaler) bool {
	for _, ownerReference := range hpa.OwnerReferences {
		gv, err := schema.ParseGroupVersion(ownerReference.APIVersion)
		if err != nil {
			return false
		}
		if gv.Group == autoscalingapi.GroupName && ownerReference.Kind == "EffectiveHorizontalPodAutoscaler" {
			return true
		}
	}
	return false
}
