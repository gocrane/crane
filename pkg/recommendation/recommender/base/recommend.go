package base

import (
	"github.com/montanaflynn/stats"

	"github.com/gocrane/crane/pkg/common"
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

func (br *BaseRecommender) GetPercentile(percentile float64, ts []*common.TimeSeries) (float64, error) {
	var values stats.Float64Data
	for _, ss := range ts[0].Samples {
		values = append(values, ss.Value)
	}
	return stats.Percentile(values, percentile)
}
