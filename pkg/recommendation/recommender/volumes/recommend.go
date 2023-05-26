package volumes

import (
	"github.com/gocrane/crane/pkg/recommendation/framework"
)

func (vr *VolumesRecommender) PreRecommend(ctx *framework.RecommendationContext) error {
	return nil
}

func (vr *VolumesRecommender) Recommend(ctx *framework.RecommendationContext) error {
	ctx.Recommendation.Status.Action = "Delete"
	ctx.Recommendation.Status.Description = "It is an Orphan Volumes"
	return nil
}

// Policy add some logic for result of recommend phase.
func (vr *VolumesRecommender) Policy(ctx *framework.RecommendationContext) error {
	return nil
}
