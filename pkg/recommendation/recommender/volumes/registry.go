package volumes

import (
	analysisv1alph1 "github.com/gocrane/api/analysis/v1alpha1"
	"github.com/gocrane/crane/pkg/recommendation/config"
	"github.com/gocrane/crane/pkg/recommendation/recommender"
	"github.com/gocrane/crane/pkg/recommendation/recommender/apis"
	"github.com/gocrane/crane/pkg/recommendation/recommender/base"
)

var _ recommender.Recommender = &VolumesRecommender{}

type VolumesRecommender struct {
	base.BaseRecommender
}

func (s *VolumesRecommender) Name() string {
	return recommender.VolumesRecommender
}

// NewVolumesRecommender create a new Volumes recommender.
func NewServiceRecommender(recommender apis.Recommender, recommendationRule analysisv1alph1.RecommendationRule) (*VolumesRecommender, error) {
	recommender = config.MergeRecommenderConfigFromRule(recommender, recommendationRule)
	return &VolumesRecommender{
		*base.NewBaseRecommender(recommender),
	}, nil
}
