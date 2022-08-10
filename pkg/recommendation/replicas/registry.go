package replicas

import (
	"github.com/gocrane/crane/pkg/recommendation/recommender"
	"github.com/gocrane/crane/pkg/recommendation/recommender/apis"
)

var _ recommender.Recommender = &ReplicasRecommender{}

type ReplicasRecommender struct {
	apis.Recommender
}

func (rr *ReplicasRecommender) Name() string {
	return recommender.ReplicasRecommender
}

// NewReplicasRecommender create a new replicas recommender.
func NewReplicasRecommender(recommender apis.Recommender) *ReplicasRecommender {
	return &ReplicasRecommender{recommender}
}
