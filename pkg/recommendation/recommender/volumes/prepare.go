package volumes

import (
	"github.com/gocrane/crane/pkg/recommendation/framework"
)

// CheckDataProviders in PrePrepare phase, will create data source provider via your recommendation config.
func (rr *VolumesRecommender) CheckDataProviders(ctx *framework.RecommendationContext) error {
	if err := rr.BaseRecommender.CheckDataProviders(ctx); err != nil {
		return err
	}

	return nil
}

func (rr *VolumesRecommender) CollectData(ctx *framework.RecommendationContext) error {
	return nil
}

func (rr *VolumesRecommender) PostProcessing(ctx *framework.RecommendationContext) error {
	return nil
}
