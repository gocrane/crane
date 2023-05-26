package service

import (
	"fmt"

	"github.com/montanaflynn/stats"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/recommendation/framework"
)

func (s *ServiceRecommender) PreRecommend(ctx *framework.RecommendationContext) error {
	return nil
}

func (s *ServiceRecommender) Recommend(ctx *framework.RecommendationContext) error {
	if len(ctx.Pods) == 0 {
		ctx.Recommendation.Status.Action = "Delete"
		ctx.Recommendation.Status.Description = "It is a Orphan Service, Pod count is 0"
		return nil
	}

	// check if pod net receive percentile bytes lt config value
	netReceiveBytes, err := s.getPercentile(s.netReceivePercentile, ctx.InputValue(netReceiveBytesKey))
	if err != nil {
		return err
	}
	if netReceiveBytes > s.netReceiveBytes {
		return fmt.Errorf("Service %s is not a Orphan Service, because the config value is %f, but the net receive %f percentile bytes is %f ",
			ctx.Object.GetName(), s.netReceiveBytes, s.netReceivePercentile, netReceiveBytes)
	}

	// check if pod net transfer percentile bytes lt config value
	netTransferBytes, err := s.getPercentile(s.netTransferPercentile, ctx.InputValue(netTransferBytesKey))
	if err != nil {
		return err
	}
	if netTransferBytes > s.netTransferBytes {
		return fmt.Errorf("Service %s is not a Orphan Service, because the config value is %f, but the net transfer %f percentile bytes is %f ",
			ctx.Object.GetName(), s.netTransferBytes, s.netTransferPercentile, netTransferBytes)
	}

	ctx.Recommendation.Status.Action = "Delete"
	ctx.Recommendation.Status.Description = "It is a Orphan Service, Pod net bytes low"
	return nil
}

// Policy add some logic for result of recommend phase.
func (s *ServiceRecommender) Policy(ctx *framework.RecommendationContext) error {
	return nil
}

func (s *ServiceRecommender) getPercentile(configPercentile float64, ts []*common.TimeSeries) (float64, error) {
	var values stats.Float64Data
	for _, ss := range ts[0].Samples {
		values = append(values, ss.Value)
	}
	return stats.Percentile(values, configPercentile)
}
