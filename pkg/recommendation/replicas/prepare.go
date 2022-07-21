package replicas

import "github.com/gocrane/crane/pkg/recommendation/framework"

func (rr *ReplicasRecommender) CheckDataProviders(ctx *framework.RecommendationContext) error {
	// 1. load data provider from recommendation config
	return nil
}

func (rr *ReplicasRecommender) CollectData(ctx *framework.RecommendationContext) error {
	return nil
}

func (rr *ReplicasRecommender) PostProcessing(ctx *framework.RecommendationContext) error {
	return nil
}
