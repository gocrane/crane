package inspector

import (
	analysisapi "github.com/gocrane/api/analysis/v1alpha1"

	"github.com/gocrane/crane/pkg/recommend/types"
)

type Inspector interface {
	// Name return name for current Interface
	Name() string

	// Inspect valid for Context to ensure the target is available for recommendation
	Inspect() error
}

func NewInspectors(ctx *types.Context) []Inspector {
	var inspectors []Inspector

	switch ctx.Recommendation.Spec.Type {
	case analysisapi.AnalysisTypeResource:
		if ctx.Pods != nil {
			inspectors = append(inspectors, &ResourceRequestInspector{Context: ctx})
		}
	case analysisapi.AnalysisTypeReplicas:
		if ctx.Scale != nil {
			inspector := &WorkloadInspector{
				Context: ctx,
			}
			inspectors = append(inspectors, inspector)
		}

		if ctx.Pods != nil {
			inspector := &WorkloadPodsInspector{
				Pods:    ctx.Pods,
				Context: ctx,
			}
			inspectors = append(inspectors, inspector)
		}
	}

	return inspectors
}
