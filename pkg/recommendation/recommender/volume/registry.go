package volume

import (
	analysisv1alph1 "github.com/gocrane/api/analysis/v1alpha1"

	"github.com/gocrane/crane/pkg/recommendation/config"
	"github.com/gocrane/crane/pkg/recommendation/recommender"
	"github.com/gocrane/crane/pkg/recommendation/recommender/apis"
	"github.com/gocrane/crane/pkg/recommendation/recommender/base"
)

var _ recommender.Recommender = &VolumeRecommender{}

type VolumeRecommender struct {
	base.BaseRecommender
}

func init() {
	recommender.RegisterRecommenderProvider(recommender.VolumeRecommender, NewVolumeRecommender)
}

func (vr *VolumeRecommender) Name() string {
	return recommender.VolumeRecommender
}

// NewVolumeRecommender create a new Volumes recommender.
func NewVolumeRecommender(recommender apis.Recommender, recommendationRule analysisv1alph1.RecommendationRule) (recommender.Recommender, error) {
	recommender = config.MergeRecommenderConfigFromRule(recommender, recommendationRule)
	return &VolumeRecommender{
		*base.NewBaseRecommender(recommender),
	}, nil
}
