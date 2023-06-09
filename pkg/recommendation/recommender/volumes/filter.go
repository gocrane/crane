package volumes

import (
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

	if err = framework.RetrievePersistentVolumeClaims(ctx); err != nil {
		return err
	}

	if len(ctx.PVCs) == 0 {
		return nil
	}

	if ctx.Pods, err = utils.GetNamespacePods(ctx.Client, ctx.PVCs[0].GetNamespace()); err != nil {
		return err
	}

	return nil
}
