package replicas

import (
	"encoding/json"
	"fmt"
	"github.com/gocrane/crane/pkg/metrics"
	"github.com/gocrane/crane/pkg/recommendation/framework"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Observe enhance the observability.
func (rr *ReplicasRecommender) Observe(ctx *framework.RecommendationContext) error {
	// get new PodTemplate
	newObject := unstructured.Unstructured{}
	if err := json.Unmarshal([]byte(ctx.Recommendation.Status.RecommendedInfo), newObject); err != nil {
		return err
	}

	replicas, found, err := unstructured.NestedFloat64(newObject.Object, "spec", "replicas")
	if !found || err != nil {
		return fmt.Errorf("get replicas from unstructed object failed. ")
	}

	labels := map[string]string{
		"apiversion": ctx.Recommendation.Spec.TargetRef.APIVersion,
		"owner_kind": ctx.Recommendation.Spec.TargetRef.Kind,
		"namespace":  ctx.Recommendation.Spec.TargetRef.Namespace,
		"owner_name": ctx.Recommendation.Spec.TargetRef.Name,
	}
	metrics.ReplicasRecommendation.With(labels).Set(replicas)

	return nil
}
