package resource

import (
	"encoding/json"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/gocrane/crane/pkg/metrics"
	"github.com/gocrane/crane/pkg/recommendation/framework"
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

	for _, container := range newPodTemplate.Spec.Containers {
		rr.recordResourceRecommendation(ctx, container.Name, v1.ResourceCPU, container.Resources.Requests[v1.ResourceCPU])
		rr.recordResourceRecommendation(ctx, container.Name, v1.ResourceMemory, container.Resources.Requests[v1.ResourceMemory])
	}

	return nil
}

func (rr *ResourceRecommender) recordResourceRecommendation(ctx *framework.RecommendationContext, containerName string, resName v1.ResourceName, quantity resource.Quantity) {
	labels := map[string]string{
		"apiversion": ctx.Recommendation.Spec.TargetRef.APIVersion,
		"owner_kind": ctx.Recommendation.Spec.TargetRef.Kind,
		"namespace":  ctx.Recommendation.Spec.TargetRef.Namespace,
		"owner_name": ctx.Recommendation.Spec.TargetRef.Name,
		"container":  containerName,
		"resource":   resName.String(),
	}

	labels["owner_replicas"] = fmt.Sprintf("%d", len(ctx.Pods))

	switch resName {
	case v1.ResourceCPU:
		metrics.ResourceRecommendation.With(labels).Set(float64(quantity.MilliValue()) / 1000.)
	case v1.ResourceMemory:
		metrics.ResourceRecommendation.With(labels).Set(float64(quantity.Value()))
	}
}
