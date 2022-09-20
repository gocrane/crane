package base

import (
	"time"

	"github.com/gocrane/crane/pkg/recommendation/recommender"
	"github.com/gocrane/crane/pkg/recommendation/recommender/apis"
)

var _ recommender.Recommender = &BaseRecommender{}

const DefaultCreationCoolDown = time.Minute * 3

type BaseRecommender struct {
	apis.Recommender
	CreationCoolDown time.Duration
}

func (br *BaseRecommender) Name() string {
	return ""
}

// NewBaseRecommender create a new base recommender.
func NewBaseRecommender(recommender apis.Recommender) *BaseRecommender {
	creationCoolDown, exists := recommender.Config["creation-cooldown"]
	creationCoolDownDuration, err := time.ParseDuration(creationCoolDown)
	if err != nil || !exists {
		creationCoolDownDuration = DefaultCreationCoolDown
	}

	return &BaseRecommender{
		recommender,
		creationCoolDownDuration,
	}
}
