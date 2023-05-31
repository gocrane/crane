package service

import (
	"fmt"

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
	if s.netReceiveBytes != 0 {
		netReceiveBytes, err := s.BaseRecommender.GetPercentile(s.netReceivePercentile, ctx.InputValue(netReceiveBytesKey))
		if err != nil {
			return err
		}
		if netReceiveBytes > s.netReceiveBytes {
			return fmt.Errorf("Service %s is not a Orphan Service, because the config value is %f, but the net receive %f percentile bytes is %f ",
				ctx.Object.GetName(), s.netReceiveBytes, s.netReceivePercentile, netReceiveBytes)
		}
	}

	// check if pod net transfer percentile bytes lt config value
	if s.netTransferBytes != 0 {
		netTransferBytes, err := s.BaseRecommender.GetPercentile(s.netTransferPercentile, ctx.InputValue(netTransferBytesKey))
		if err != nil {
			return err
		}
		if netTransferBytes > s.netTransferBytes {
			return fmt.Errorf("Service %s is not a Orphan Service, because the config value is %f, but the net transfer %f percentile bytes is %f ",
				ctx.Object.GetName(), s.netTransferBytes, s.netTransferPercentile, netTransferBytes)
		}
	}

	ctx.Recommendation.Status.Action = "Delete"
	ctx.Recommendation.Status.Description = "It is a Orphan Service, Pod net bytes low"
	return nil
}

// Policy add some logic for result of recommend phase.
func (s *ServiceRecommender) Policy(ctx *framework.RecommendationContext) error {
	return nil
}
