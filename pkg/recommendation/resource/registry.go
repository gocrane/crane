package resource

import (
	"github.com/gocrane/crane/pkg/recommendation/recommender"
	"github.com/gocrane/crane/pkg/recommendation/recommender/apis"
)

var _ recommender.Recommender = &ResourceRecommender{}

type ResourceRecommender struct {
	apis.Recommender
}

func (rr *ResourceRecommender) Name() string {
	return recommender.ResourceRecommender
}

// NewResourceRecommender create a new resource recommender.
func NewResourceRecommender(recommender apis.Recommender) *ResourceRecommender {
	return &ResourceRecommender{recommender}
}
