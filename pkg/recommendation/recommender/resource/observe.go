package resource

import (
	"context"
	"encoding/json"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/gocrane/crane/pkg/metrics"
	"github.com/gocrane/crane/pkg/recommendation/framework"
	"github.com/gocrane/crane/pkg/utils"
)

// Observe enhance the observability.
func (rr *ResourceRecommender) Observe(ctx *framework.RecommendationContext) error {
	// get new PodTemplate
	var newObject map[string]interface{}
	if err := json.Unmarshal([]byte(ctx.Recommendation.Status.RecommendedInfo), &newObject); err != nil {
		return err
	}

	podTemplateObject, found, err := unstructured.NestedMap(newObject, "spec", "template")
	if !found || err != nil {
		return fmt.Errorf("get template from unstructed object failed. ")
	}

	var newPodTemplate v1.PodTemplateSpec
	err = framework.ObjectConversion(podTemplateObject, &newPodTemplate)
	if err != nil {
		return err
	}

	resourceNameList := []v1.ResourceName{v1.ResourceCPU, v1.ResourceMemory}
	for _, container := range newPodTemplate.Spec.Containers {
		for _, resourceName := range resourceNameList {
			err = rr.recordResourceRecommendation(ctx, container.Name, resourceName, container.Resources.Requests[resourceName])
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (rr *ResourceRecommender) recordResourceRecommendation(ctx *framework.RecommendationContext, containerName string, resName v1.ResourceName, quantity resource.Quantity) error {
	labels := map[string]string{
		"apiversion": ctx.Recommendation.Spec.TargetRef.APIVersion,
		"owner_kind": ctx.Recommendation.Spec.TargetRef.Kind,
		"namespace":  ctx.Recommendation.Spec.TargetRef.Namespace,
		"owner_name": ctx.Recommendation.Spec.TargetRef.Name,
		"container":  containerName,
		"resource":   resName.String(),
	}

	scale, _, err := utils.GetScaleFromObjectReference(context.TODO(), ctx.RestMapper, ctx.ScaleClient, ctx.Recommendation.Spec.TargetRef)
	if err != nil {
		return err
	}
	labels["owner_replicas"] = fmt.Sprintf("%d", scale.Spec.Replicas)

	switch resName {
	case v1.ResourceCPU:
		metrics.ResourceRecommendation.With(labels).Set(float64(quantity.MilliValue()) / 1000.)
	case v1.ResourceMemory:
		metrics.ResourceRecommendation.With(labels).Set(float64(quantity.Value()))
	}

	return nil
}
