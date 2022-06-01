package advisor

import (
	analysisapi "github.com/gocrane/api/analysis/v1alpha1"

	"github.com/gocrane/crane/pkg/recommend/types"
)

type Advisor interface {
	// Name return name for current Interface
	Name() string

	// Advise analysis and give advice in ProposedRecommendation
	Advise(proposed *types.ProposedRecommendation) error
}

func NewAdvisors(ctx *types.Context) (advisors []Advisor) {
	switch ctx.Recommendation.Spec.Type {
	case analysisapi.AnalysisTypeResource:
		advisors = []Advisor{
			&ResourceRequestAdvisor{
				Context: ctx,
			},
		}
	case analysisapi.AnalysisTypeReplicas:
		advisors = []Advisor{
			&ReplicasAdvisor{
				Context: ctx,
			},
		}
	}

	return
}
