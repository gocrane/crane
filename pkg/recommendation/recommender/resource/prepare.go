package resource

import (
	"github.com/gocrane/crane/pkg/recommendation/framework"
)

// CheckDataProviders in PrePrepare phase, will create data source provider via your recommendation config.
func (rr *ResourceRecommender) CheckDataProviders(ctx *framework.RecommendationContext) error {
	if err := rr.BaseRecommender.CheckDataProviders(ctx); err != nil {
		return err
	}

	return nil
}

func (rr *ResourceRecommender) CollectData(ctx *framework.RecommendationContext) error {
	return nil
}

func (rr *ResourceRecommender) PostProcessing(ctx *framework.RecommendationContext) error {
	return nil
}
