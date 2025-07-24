package hpa

import (
	"github.com/gocrane/crane/pkg/recommendation/framework"
)

// CheckDataProviders in PrePrepare phase, will create data source provider via your recommendation config.
func (rr *HPARecommender) CheckDataProviders(ctx *framework.RecommendationContext) error {
	return rr.ReplicasRecommender.CheckDataProviders(ctx)
}

func (rr *HPARecommender) CollectData(ctx *framework.RecommendationContext) error {
	return rr.ReplicasRecommender.CollectData(ctx)
}

func (rr *HPARecommender) PostProcessing(ctx *framework.RecommendationContext) error {
	return rr.ReplicasRecommender.PostProcessing(ctx)
}
