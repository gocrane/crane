package resource

import (
	"fmt"

	"github.com/gocrane/crane/pkg/recommendation/framework"
)

// Filter out k8s resources that are not supported by the recommender.
func (rr *ResourceRecommender) Filter(ctx *framework.RecommendationContext) error {
	var err error

	// filter resource that not match objectIdentity
	if err = rr.BaseRecommender.Filter(ctx); err != nil {
		return err
	}

	if err = framework.RetrievePodTemplate(ctx); err != nil {
		return err
	}

	if err = framework.RetrieveScale(ctx); err != nil {
		return err
	}

	if err = framework.RetrievePods(ctx); err != nil {
		return err
	}

	// filter workloads that are downing
	if len(ctx.Pods) == 0 {
		return fmt.Errorf("pod not found")
	}

	pod := ctx.Pods[0]
	if ctx.Recommendation.Spec.TargetRef.Kind == "Pod" {
		if len(pod.OwnerReferences) != 0 {
			return fmt.Errorf("when Kind == 'Pod', only recommend pods have no owner reference")
		}
	} else {
		if len(pod.OwnerReferences) == 0 {
			return fmt.Errorf("owner reference not found")
		}
	}

	return nil
}
