package replicas

import (
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/gocrane/crane/pkg/recommend/types"
	"github.com/gocrane/crane/pkg/recommendation/framework"
)

// Observe enhance the observability.
func (rr *ReplicasRecommender) Observe(ctx *framework.RecommendationContext) error {
	key := client.ObjectKey{
		Name:      ctx.Identity.Name,
		Namespace: ctx.Identity.Namespace,
	}
	unstructed := &unstructured.Unstructured{}
	unstructed.SetAPIVersion(ctx.Identity.APIVersion)
	unstructed.SetKind(ctx.Identity.Kind)
	err := ctx.Client.Get(ctx.Context, key, unstructed)
	if err != nil {
		return err
	}

	oldObject, found, err := unstructured.NestedMap(unstructed.Object, "spec")
	if !found || err != nil {
		return fmt.Errorf("get spec from unstructed object %s failed. ", klog.KObj(unstructed))
	}

	result := ctx.Recommendation.Status.RecommendedValue
	proposedRecommendation := types.ProposedRecommendation{}
	err = yaml.Unmarshal([]byte(result), proposedRecommendation)
	if err != nil {
		return fmt.Errorf("decode replicas value from context error %s. ", err)
	}

	err = unstructured.SetNestedField(unstructed.Object, proposedRecommendation.ReplicasRecommendation.Replicas, "spec", "replicas")
	if err != nil {
		return fmt.Errorf("set replicas to spec failed %s. ", err)
	}
	newObject, found, err := unstructured.NestedMap(unstructed.Object, "spec")
	if !found || err != nil {
		return fmt.Errorf("get spec from unstructed object %s failed. ", klog.KObj(unstructed))
	}

	oldBytes, err := json.Marshal(oldObject)
	if err != nil {
		return fmt.Errorf("encode error %s. ", err)
	}

	newBytes, err := json.Marshal(newObject)
	if err != nil {
		return fmt.Errorf("encode error %s. ", err)
	}

	newPatch, err := jsonpatch.CreateMergePatch(newBytes, oldBytes)
	if err != nil {
		return fmt.Errorf("create merge patch error %s. ", err)
	}
	oldPatch, err := jsonpatch.CreateMergePatch(oldBytes, newBytes)
	if err != nil {
		return fmt.Errorf("create merge patch error %s. ", err)
	}

	ctx.Recommendation.Status.RecommendedInfo = string(newPatch)
	ctx.Recommendation.Status.CurrentInfo = string(oldPatch)
	// TODO(qmhu) Create action type.
	ctx.Recommendation.Status.Action = "Patch"

	return nil
}
