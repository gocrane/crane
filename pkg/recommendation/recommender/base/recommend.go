package base

import (
	"github.com/gocrane/crane/pkg/recommendation/framework"
)

func (br *BaseRecommender) PreRecommend(ctx *framework.RecommendationContext) error {
	return nil
}

func (br *BaseRecommender) Recommend(ctx *framework.RecommendationContext) error {
	return nil
}

// Policy add some logic for result of recommend phase.
func (br *BaseRecommender) Policy(ctx *framework.RecommendationContext) error {
	return nil
}
