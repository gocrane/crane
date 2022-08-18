package base

import (
	"github.com/gocrane/crane/pkg/recommendation/framework"
)

// Observe enhance the observability.
func (br *BaseRecommender) Observe(ctx *framework.RecommendationContext) error {
	return nil
}
