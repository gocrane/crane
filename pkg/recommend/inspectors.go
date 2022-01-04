package recommend

import (
	"fmt"

	analysisapi "github.com/gocrane/api/analysis/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

type ResourceRequestInspector struct {
	*Context
}

func (i *ResourceRequestInspector) Inspect() error {
	if len(i.Pods) == 0 {
		return fmt.Errorf("pod not found")
	}

	pod := i.Pods[0]
	if len(pod.OwnerReferences) == 0 {
		return fmt.Errorf("owner reference not found")
	}

	return nil
}

type WorkloadInspector struct {
	Context *Context
}

func (i *WorkloadInspector) Inspect() error {
	return nil
}

type WorkloadPodsInspector struct {
	Context *Context
	Pods    []v1.Pod
}

func (i *WorkloadPodsInspector) Inspect() error {
	return nil
}

func NewInspectors(ctx *Context) []Inspector {
	var inspectors []Inspector

	switch ctx.Recommendation.Spec.Type {
	case analysisapi.AnalysisTypeResource:
		if ctx.Pods != nil {
			inspectors = append(inspectors, &ResourceRequestInspector{Context: ctx})
		}
	case analysisapi.AnalysisTypeHPA:
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
