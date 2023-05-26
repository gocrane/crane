package service

import (
	analysisv1alph1 "github.com/gocrane/api/analysis/v1alpha1"
	"github.com/gocrane/crane/pkg/recommendation/config"
	"github.com/gocrane/crane/pkg/recommendation/recommender"
	"github.com/gocrane/crane/pkg/recommendation/recommender/apis"
	"github.com/gocrane/crane/pkg/recommendation/recommender/base"
)

const (
	netReceiveBytesKey       = "net-receive-bytes"
	netTransferBytesKey      = "net-transfer-bytes"
	netReceivePercentileKey  = "net-receive-percentile"
	netTransferPercentileKey = "net-transfer-percentile"
)

var _ recommender.Recommender = &ServiceRecommender{}

type ServiceRecommender struct {
	base.BaseRecommender
	netReceiveBytes       float64
	netTransferBytes      float64
	netReceivePercentile  float64
	netTransferPercentile float64
}

func init() {
	recommender.RegisterRecommenderProvider(recommender.ServiceRecommender, NewServiceRecommender)
}

func (s *ServiceRecommender) Name() string {
	return recommender.ServiceRecommender
}

// NewServiceRecommender create a new service recommender.
func NewServiceRecommender(recommender apis.Recommender, recommendationRule analysisv1alph1.RecommendationRule) (recommender.Recommender, error) {
	recommender = config.MergeRecommenderConfigFromRule(recommender, recommendationRule)

	netReceiveBytes, err := recommender.GetConfigFloat(netReceiveBytesKey, 0)
	if err != nil {
		return nil, err
	}

	netTransferBytes, err := recommender.GetConfigFloat(netTransferBytesKey, 0)
	if err != nil {
		return nil, err
	}

	netReceivePercentile, err := recommender.GetConfigFloat(netReceivePercentileKey, 0)
	if err != nil {
		return nil, err
	}

	netTransferPercentile, err := recommender.GetConfigFloat(netTransferPercentileKey, 0)
	if err != nil {
		return nil, err
	}

	return &ServiceRecommender{
		*base.NewBaseRecommender(recommender),
		netReceiveBytes,
		netTransferBytes,
		netReceivePercentile,
		netTransferPercentile,
	}, nil
}
