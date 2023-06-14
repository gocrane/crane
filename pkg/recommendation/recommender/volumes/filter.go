package volumes

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/gocrane/crane/pkg/recommendation/framework"
	"github.com/gocrane/crane/pkg/utils"
)

// Filter out k8s resources that are not supported by the recommender.
func (vr *VolumesRecommender) Filter(ctx *framework.RecommendationContext) error {
	var err error

	// filter resource that not match objectIdentity
	if err = vr.BaseRecommender.Filter(ctx); err != nil {
		return err
	}

	var pv corev1.PersistentVolume
	if err = framework.ObjectConversion(ctx.Object, &pv); err != nil {
		return err
	}

	if pv.Spec.ClaimRef == nil {
		return nil
	}

	if ctx.Pods, err = utils.GetNamespacePods(ctx.Client, pv.Spec.ClaimRef.Namespace); err != nil {
		return err
	}

	return nil
}
