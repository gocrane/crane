package recommend

import (
	"fmt"

	analysisapi "github.com/gocrane/api/analysis/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	cpuQueryExprTemplate = `irate(container_cpu_usage_seconds_total{container="%s",namespace="%s",pod=~"^%s.*$"}[3m])`
	memQueryExprTemplate = `container_memory_working_set_bytes{container="%s",namespace="%s",pod=~"^%s.*$"}`
)

type MinMaxReplicasAdvisor struct {
	Context *Context
}

type ResourceRequestAdvisor struct {
	*Context
}

func (a *ResourceRequestAdvisor) Advise(proposed *ProposedRecommendation) error {
	r := &analysisapi.ResourceRequestRecommendation{}

	p := a.Predictors[predictionapi.AlgorithmTypePercentile]

	pod := a.Pods[0]
	namespace := pod.Namespace
	podNamePrefix := pod.OwnerReferences[0].Name + "-"
	for _, c := range pod.Spec.Containers {
		cr := analysisapi.ContainerRecommendation{
			ContainerName: c.Name,
			Target:        map[corev1.ResourceName]resource.Quantity{},
		}

		ts, err := p.QueryRealtimePredictedValues(fmt.Sprintf(cpuQueryExprTemplate, c.Name, namespace, podNamePrefix))
		if err != nil {
			return err
		}
		if len(ts) < 1 || len(ts[0].Samples) < 1 {
			return fmt.Errorf("no value")
		}
		v := int64(ts[0].Samples[0].Value * 1000)
		cr.Target[corev1.ResourceCPU] = *resource.NewMilliQuantity(v, resource.DecimalSI)

		ts, err = p.QueryRealtimePredictedValues(fmt.Sprintf(memQueryExprTemplate, c.Name, namespace, podNamePrefix))
		if err != nil {
			return err
		}
		if len(ts) < 1 || len(ts[0].Samples) < 1 {
			return fmt.Errorf("no value")
		}
		v = int64(ts[0].Samples[0].Value)
		cr.Target[corev1.ResourceMemory] = *resource.NewMilliQuantity(v, resource.BinarySI)
	}

	proposed.ResourceRequest = r
	return nil
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

func NewAdvisors(ctx *Context) (advisors []Advisor) {
	switch ctx.Recommendation.Spec.Type {
	case analysisapi.AnalysisTypeResource:

		advisors = []Advisor{
			&ResourceRequestAdvisor{Context: ctx},
		}
	case analysisapi.AnalysisTypeHPA:
		advisors = []Advisor{
			&MinMaxReplicasAdvisor{
				Context: ctx,
			},
			&PredictionAdvisor{
				Context: ctx,
			},
		}
	}

	return
}
