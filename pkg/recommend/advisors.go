package recommend

import (
	"fmt"

	"github.com/gocrane/crane/pkg/prediction/config"

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

func (a *ResourceRequestAdvisor) Init() error {
	p := a.Predictors[predictionapi.AlgorithmTypePercentile]

	if len(a.Pods) == 0 {
		return fmt.Errorf("pod not found")
	}
	pod := a.Pods[0]
	namespace := pod.Namespace
	podNamePrefix := pod.OwnerReferences[0].Name + "-"

	var expr string
	mc := &config.MetricContext{}
	for _, c := range pod.Spec.Containers {
		expr = fmt.Sprintf(cpuQueryExprTemplate, c.Name, namespace, podNamePrefix)
		a.V(5).Info("CPU query:", "expr", expr)
		if err := p.WithQuery(expr); err != nil {
			return err
		}
		mc.WithConfig(makeCpuConfig(expr))

		expr = fmt.Sprintf(memQueryExprTemplate, c.Name, namespace, podNamePrefix)
		a.V(5).Info("Memory query:", "expr", expr)
		if err := p.WithQuery(expr); err != nil {
			return err
		}
		mc.WithConfig(makeMemConfig(expr))
	}

	return nil
}

func makeCpuConfig(expr string) *config.Config {
	return &config.Config{
		Expression: &predictionapi.ExpressionQuery{Expression: expr},
		Percentile: &predictionapi.Percentile{
			SampleInterval: "1m",
			MarginFraction: "0.15",
			Percentile:     "0.99",
			Histogram: predictionapi.HistogramConfig{
				HalfLife:   "24h",
				BucketSize: "0.1",
				MaxValue:   "100",
			},
		},
	}
}

func makeMemConfig(expr string) *config.Config {
	return &config.Config{
		Expression: &predictionapi.ExpressionQuery{Expression: expr},
		Percentile: &predictionapi.Percentile{
			SampleInterval: "1m",
			MarginFraction: "0.15",
			Percentile:     "0.99",
			Histogram: predictionapi.HistogramConfig{
				HalfLife:   "48h",
				BucketSize: "104857600",
				MaxValue:   "104857600000",
			},
		},
	}
}

func (a *ResourceRequestAdvisor) Advise(proposed *ProposedRecommendation) error {
	r := &analysisapi.ResourceRequestRecommendation{}

	p := a.Predictors[predictionapi.AlgorithmTypePercentile]

	pod := a.Pods[0]
	namespace := pod.Namespace
	podNamePrefix := pod.OwnerReferences[0].Name + "-"

	var expr string
	for _, c := range pod.Spec.Containers {
		cr := analysisapi.ContainerRecommendation{
			ContainerName: c.Name,
			Target:        map[corev1.ResourceName]resource.Quantity{},
		}

		expr = fmt.Sprintf(cpuQueryExprTemplate, c.Name, namespace, podNamePrefix)
		ts, err := p.QueryRealtimePredictedValues(expr)
		if err != nil {
			return err
		}
		if len(ts) < 1 || len(ts[0].Samples) < 1 {
			return fmt.Errorf("no value, expr: %s", expr)
		}
		v := int64(ts[0].Samples[0].Value * 1000)
		cr.Target[corev1.ResourceCPU] = *resource.NewMilliQuantity(v, resource.DecimalSI)

		expr = fmt.Sprintf(memQueryExprTemplate, c.Name, namespace, podNamePrefix)
		ts, err = p.QueryRealtimePredictedValues(expr)
		if err != nil {
			return err
		}
		if len(ts) < 1 || len(ts[0].Samples) < 1 {
			return fmt.Errorf("no value, expr: %s", expr)
		}
		v = int64(ts[0].Samples[0].Value)
		cr.Target[corev1.ResourceMemory] = *resource.NewMilliQuantity(v, resource.BinarySI)

		r.Containers = append(r.Containers, cr)
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
		a := &ResourceRequestAdvisor{Context: ctx}
		if err := a.Init(); err != nil {
			panic(err)
		}
		advisors = []Advisor{a}
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
