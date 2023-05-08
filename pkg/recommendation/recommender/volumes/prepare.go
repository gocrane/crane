package volumes

import (
	"github.com/gocrane/crane/pkg/recommendation/framework"
)

// CheckDataProviders in PrePrepare phase, will create data source provider via your recommendation config.
func (vr *VolumesRecommender) CheckDataProviders(ctx *framework.RecommendationContext) error {
	if err := vr.BaseRecommender.CheckDataProviders(ctx); err != nil {
		return err
	}

	return nil
}

func (vr *VolumesRecommender) CollectData(ctx *framework.RecommendationContext) error {
	return nil
}

func (vr *VolumesRecommender) PostProcessing(ctx *framework.RecommendationContext) error {
	return nil
}
