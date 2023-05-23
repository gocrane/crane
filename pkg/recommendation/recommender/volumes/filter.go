package volumes

import (
	"github.com/gocrane/crane/pkg/recommendation/framework"
)

// Filter out k8s resources that are not supported by the recommender.
func (vr *VolumesRecommender) Filter(ctx *framework.RecommendationContext) error {
	var err error

	// filter resource that not match objectIdentity
	if err = vr.BaseRecommender.Filter(ctx); err != nil {
		return err
	}

	if err = framework.RetrieveVolumes(ctx); err != nil {
		return err
	}

	return nil
}
