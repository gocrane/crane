package base

import (
	"github.com/gocrane/crane/pkg/recommendation/recommender"
	"github.com/gocrane/crane/pkg/recommendation/recommender/apis"
)

var _ recommender.Recommender = &BaseRecommender{}

type BaseRecommender struct {
	apis.Recommender
}

func (br *BaseRecommender) Name() string {
	return recommender.ReplicasRecommender
}

// NewBaseRecommender create a new base recommender.
func NewBaseRecommender(recommender apis.Recommender) *BaseRecommender {
	return &BaseRecommender{recommender}
}
