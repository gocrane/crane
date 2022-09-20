package idlenode

import (
	"github.com/gocrane/crane/pkg/recommendation/framework"
)

// CheckDataProviders in PrePrepare phase, will create data source provider via your recommendation config.
func (inr *IdleNodeRecommender) CheckDataProviders(ctx *framework.RecommendationContext) error {
	if err := inr.BaseRecommender.CheckDataProviders(ctx); err != nil {
		return err
	}

	return nil
}

func (inr *IdleNodeRecommender) CollectData(ctx *framework.RecommendationContext) error {
	return nil
}

func (inr *IdleNodeRecommender) PostProcessing(ctx *framework.RecommendationContext) error {
	return nil
}
