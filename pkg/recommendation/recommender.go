package recommendation

import (
	"github.com/gocrane/crane/pkg/recommendation/framework"
)

type Recommender interface {
	Name() RecommenderName
	Run(ctx *framework.RecommendationContext)
	framework.Filter
	framework.PrePrepare
	framework.Prepare
	framework.PostPrepare
	framework.Recommend
	framework.PostRecommend
	framework.Observe
}
