package utils

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	autoscalingapiv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/scale"
)

func GetScale(ctx context.Context, restMapper meta.RESTMapper, scaleClient scale.ScalesGetter, namespace string, ref autoscalingv2.CrossVersionObjectReference) (*autoscalingapiv1.Scale, *meta.RESTMapping, error) {
	targetGV, err := schema.ParseGroupVersion(ref.APIVersion)
	if err != nil {
		return nil, nil, err
	}

	targetGK := schema.GroupKind{
		Group: targetGV.Group,
		Kind:  ref.Kind,
	}

	mappings, err := restMapper.RESTMappings(targetGK)
	if err != nil {
		return nil, nil, err
	}

	for _, mapping := range mappings {
		scale, err := scaleClient.Scales(namespace).Get(ctx, mapping.Resource.GroupResource(), ref.Name, metav1.GetOptions{})
		if err == nil {
			return scale, mapping, nil
		}
	}

	return nil, nil, fmt.Errorf("unrecognized resource")
}

func GetPodsFromScale(kubeClient client.Client, scale *autoscalingapiv1.Scale) ([]v1.Pod, error) {
	selector, err := labels.ConvertSelectorToLabelsMap(scale.Status.Selector)
	if err != nil {
		return nil, err
	}

	opts := []client.ListOption{
		client.InNamespace(scale.GetNamespace()),
		client.MatchingLabels(selector),
	}

	podList := &v1.PodList{}
	err = kubeClient.List(context.TODO(), podList, opts...)
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}
