package hpa

import (
	"context"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"

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
			hpa.Spec.ScaleTargetRef.Kind == ctx.Recommendation.Spec.TargetRef.Kind &&
			hpa.Spec.ScaleTargetRef.APIVersion == ctx.Recommendation.Spec.TargetRef.APIVersion {
			ctx.HPA = &hpa
			break
		}
	}

	ehpaList := &autoscalingapi.EffectiveHorizontalPodAutoscalerList{}
	opts = []client.ListOption{
		client.InNamespace(ctx.Recommendation.Spec.TargetRef.Namespace),
	}
	err = ctx.Client.List(context.TODO(), ehpaList, opts...)
	if err != nil {
		return err
	}

	for _, ehpa := range ehpaList.Items {
		if ehpa.Spec.ScaleTargetRef.Name == ctx.Recommendation.Spec.TargetRef.Name &&
			ehpa.Spec.ScaleTargetRef.Kind == ctx.Recommendation.Spec.TargetRef.Kind &&
			ehpa.Spec.ScaleTargetRef.APIVersion == ctx.Recommendation.Spec.TargetRef.APIVersion {
			ctx.EHPA = &ehpa
			break
		}
	}

	return nil
}
