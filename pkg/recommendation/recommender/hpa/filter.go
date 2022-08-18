package hpa

import (
	"context"

	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/recommendation/framework"
)

// Filter out k8s resources that are not supported by the recommender.
func (rr *HPARecommender) Filter(ctx *framework.RecommendationContext) error {
	if err := rr.ReplicasRecommender.Filter(ctx); err != nil {
		return err
	}

	hpaList := &autoscalingv2.HorizontalPodAutoscalerList{}
	opts := []client.ListOption{
		client.InNamespace(ctx.Recommendation.Spec.TargetRef.Namespace),
	}
	err := ctx.Client.List(context.TODO(), hpaList, opts...)
	if err != nil {
		return err
	}

	for _, hpa := range hpaList.Items {
		// bypass hpa that controller by ehpa
		if hpa.Labels != nil && hpa.Labels["app.kubernetes.io/managed-by"] == known.EffectiveHorizontalPodAutoscalerManagedBy {
			continue
		}

		if hpa.Spec.ScaleTargetRef.Name == ctx.Recommendation.Spec.TargetRef.Name &&
			hpa.Spec.ScaleTargetRef.Kind == ctx.Recommendation.Spec.TargetRef.APIVersion &&
			hpa.Spec.ScaleTargetRef.APIVersion == ctx.Recommendation.Spec.TargetRef.APIVersion {
			ctx.HPA = &hpa
		}
	}

	return nil
}
