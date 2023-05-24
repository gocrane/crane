package volumes

import (
	"github.com/gocrane/crane/pkg/recommendation/framework"
)

func (s *VolumesRecommender) PreRecommend(ctx *framework.RecommendationContext) error {
	return nil
}

func (s *VolumesRecommender) Recommend(ctx *framework.RecommendationContext) error {
	ctx.Recommendation.Status.Action = "Delete"
	ctx.Recommendation.Status.Description = "It is an Orphan Volumes"
	return nil
}

// Policy add some logic for result of recommend phase.
func (s *VolumesRecommender) Policy(ctx *framework.RecommendationContext) error {
	return nil
}
