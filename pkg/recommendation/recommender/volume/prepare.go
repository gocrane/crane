package volume

import (
	"github.com/gocrane/crane/pkg/recommendation/framework"
)

// CheckDataProviders in PrePrepare phase, will create data source provider via your recommendation config.
func (vr *VolumeRecommender) CheckDataProviders(ctx *framework.RecommendationContext) error {
	if err := vr.BaseRecommender.CheckDataProviders(ctx); err != nil {
		return err
	}

	return nil
}

func (vr *VolumeRecommender) CollectData(ctx *framework.RecommendationContext) error {
	return nil
}

func (vr *VolumeRecommender) PostProcessing(ctx *framework.RecommendationContext) error {
	return nil
}
