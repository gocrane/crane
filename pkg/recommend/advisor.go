package recommend

import analysisapi "github.com/gocrane/api/analysis/v1alpha1"

type MinMaxReplicasAdvisor struct {
	Context *Context
}

func (a *MinMaxReplicasAdvisor) Advise(proposed *ProposedRecommendation) error {
	return nil
}

type PredictionAdvisor struct {
	Context *Context
}

func (a *PredictionAdvisor) Advise(proposed *ProposedRecommendation) error {
	return nil
}

func NewAdvisors(context *Context) (advisors []Advisor) {
	switch context.Recommendation.Spec.Type {
	case analysisapi.AnalysisTypeResource:
		// todo
	case analysisapi.AnalysisTypeHPA:
		advisors = []Advisor{
			&MinMaxReplicasAdvisor{
				Context: context,
			},
			&PredictionAdvisor{
				Context: context,
			},
		}
	}

	return
}
