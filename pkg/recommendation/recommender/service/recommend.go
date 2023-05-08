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

	// check if pod net receive bytes lt config value
	if netReceiveBytes := s.getMaxValue(s.netReceiveBytes, ctx.InputValue(netReceiveBytesKey)); netReceiveBytes > s.netReceiveBytes {
		return fmt.Errorf("Service %s is not a Orphan Service, because the config value is %f, but the net receive bytes is %f ", ctx.Object.GetName(), s.netReceiveBytes, netReceiveBytes)
	}

	// check if pod net transfer bytes lt config value
	if netTransferBytes := s.getMaxValue(s.netTransferBytes, ctx.InputValue(netTransferBytesKey)); netTransferBytes > s.netTransferBytes {
		return fmt.Errorf("Service %s is not a Orphan Service, because the config value is %f, but the net transfer bytes is %f ", ctx.Object.GetName(), s.netTransferBytes, netTransferBytes)
	}

	// check if pod net receive percentile lt config value
	if netReceivePercentile := s.getPercentile(s.netTransferBytes, ctx.InputValue(netReceiveBytesKey)); netReceivePercentile > s.netReceivePercentile {
		return fmt.Errorf("Service %s is not a Orphan Service, because the config value is %f, but the net receive percentile is %f ", ctx.Object.GetName(), s.netReceivePercentile, netReceivePercentile)
	}

	// check if pod net transfer percentile lt config value
	if netTransferPercentile := s.getPercentile(s.netTransferBytes, ctx.InputValue(netTransferBytesKey)); netTransferPercentile > s.netTransferPercentile {
		return fmt.Errorf("Service %s is not a Orphan Service, because the config value is %f, but the net transfer percentile is %f ", ctx.Object.GetName(), s.netTransferPercentile, netTransferPercentile)
	}

	ctx.Recommendation.Status.Action = "Delete"
	ctx.Recommendation.Status.Description = "It is a Orphan Service, Pod net bytes low"
	return nil
}

// Policy add some logic for result of recommend phase.
func (s *ServiceRecommender) Policy(ctx *framework.RecommendationContext) error {
	return nil
}

func (s *ServiceRecommender) getMaxValue(configValue float64, ts []*common.TimeSeries) float64 {
	if configValue == 0 {
		return configValue
	}
	var maxValue float64
	for _, ss := range ts[0].Samples {
		if ss.Value > maxValue {
			maxValue = ss.Value
		}
	}
	return maxValue
}

func (s *ServiceRecommender) getPercentile(configValue float64, ts []*common.TimeSeries) float64 {
	if configValue == 0 {
		return configValue
	}
	var input stats.Float64Data
	for _, ss := range ts[0].Samples {
		input = append(input, ss.Value)
	}
	percentile, err := stats.Percentile(input, configValue)
	if err != nil {
		return configValue
	}
	return percentile
}
