package recommender

import (
	"github.com/gocrane/crane/pkg/recommendation/framework"
)

type Recommender interface {
	Name() string
	framework.Filter
	framework.PrePrepare
	framework.Prepare
	framework.PostPrepare
	framework.PreRecommend
	framework.Recommend
	framework.PostRecommend
	framework.Observe
}
