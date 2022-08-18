package replicas

import (
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/gocrane/crane/pkg/metrics"
	"github.com/gocrane/crane/pkg/recommendation/framework"
)

// Observe enhance the observability.
func (rr *ReplicasRecommender) Observe(ctx *framework.RecommendationContext) error {
	if ctx.Recommendation.Status.Action == "Patch" {
		// get new PodTemplate
		var newObject map[string]interface{}
		if err := json.Unmarshal([]byte(ctx.Recommendation.Status.RecommendedInfo), &newObject); err != nil {
			return err
		}

		replicas, found, err := unstructured.NestedFloat64(newObject, "spec", "replicas")
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

	}

	return nil
}
