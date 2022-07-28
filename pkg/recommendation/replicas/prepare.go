package replicas

import (
	"github.com/gocrane/crane/pkg/providers"
	"github.com/gocrane/crane/pkg/providers/prom"
	"github.com/gocrane/crane/pkg/recommendation/framework"
	"strings"
)

func (rr *ReplicasRecommender) CheckDataProviders(ctx *framework.RecommendationContext) error {
	// 1. load data provider from recommendation config, override the default data source
	configSet := rr.Recommender.Config
	for name, value := range configSet {
		switch strings.ToLower(name) {
		case string(providers.PrometheusDataSource):
			promConfig := providers.PromConfig{}
			provider, err := prom.NewProvider()
		}
	}
	// 2. if not set data provider, will use default
	return nil
}

func (rr *ReplicasRecommender) CollectData(ctx *framework.RecommendationContext) error {
	return nil
}

func (rr *ReplicasRecommender) PostProcessing(ctx *framework.RecommendationContext) error {
	return nil
}
