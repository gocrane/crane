package advisor

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"

	analysisapi "github.com/gocrane/api/analysis/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/recommend/types"
)

const (
	cpuQueryExprTemplate = `irate(container_cpu_usage_seconds_total{container="%s",namespace="%s",pod=~"^%s.*$"}[3m])`
	memQueryExprTemplate = `container_memory_working_set_bytes{container="%s",namespace="%s",pod=~"^%s.*$"}`
)

const (
	DefaultNamespace = "default"
)

type ResourceRequestAdvisor struct {
	*types.Context
}

func makeCpuConfig(expr string, props map[string]string) *config.Config {
	sampleInterval, exists := props["cpu-sample-interval"]
	if !exists {
		sampleInterval = "1m"
	}
	percentile, exists := props["cpu-request-percentile"]
	if !exists {
		percentile = "0.99"
	}
	marginFraction, exists := props["cpu-request-margin-fraction"]
	if !exists {
		marginFraction = "0.15"
	}

	return &config.Config{
		Expression: &predictionapi.ExpressionQuery{Expression: expr},
		Percentile: &predictionapi.Percentile{
			SampleInterval: sampleInterval,
			MarginFraction: marginFraction,
			Percentile:     percentile,
			Histogram: predictionapi.HistogramConfig{
				HalfLife:   "24h",
				BucketSize: "0.1",
				MaxValue:   "100",
			},
		},
	}
}

func makeMemConfig(expr string, props map[string]string) *config.Config {
	sampleInterval, exists := props["mem-sample-interval"]
	if !exists {
		sampleInterval = "1m"
	}
	percentile, exists := props["mem-request-percentile"]
	if !exists {
		percentile = "0.99"
	}
	marginFraction, exists := props["mem-request-margin-fraction"]
	if !exists {
		marginFraction = "0.15"
	}

	return &config.Config{
		Expression: &predictionapi.ExpressionQuery{Expression: expr},
		Percentile: &predictionapi.Percentile{
			SampleInterval: sampleInterval,
			MarginFraction: marginFraction,
			Percentile:     percentile,
			Histogram: predictionapi.HistogramConfig{
				HalfLife:   "48h",
				BucketSize: "104857600",
				MaxValue:   "104857600000",
			},
		},
	}
}

func (a *ResourceRequestAdvisor) Advise(proposed *types.ProposedRecommendation) error {
	r := &analysisapi.ResourceRequestRecommendation{}

	p := a.Predictors[predictionapi.AlgorithmTypePercentile]

	if len(a.Pods) == 0 {
		return fmt.Errorf("pod not found")
	}

	mc := &config.MetricContext{}
	pod := a.Pods[0]
	namespace := pod.Namespace
	podNamePrefix := pod.OwnerReferences[0].Name + "-"

	var expr string
	for _, c := range pod.Spec.Containers {
		expr = fmt.Sprintf(cpuQueryExprTemplate, c.Name, namespace, podNamePrefix)
		klog.V(4).Infof("CPU query: %s", expr)
		if err := p.WithQuery(expr); err != nil {
			return err
		}
		mc.WithConfig(makeCpuConfig(expr, a.ConfigProperties))

		expr = fmt.Sprintf(memQueryExprTemplate, c.Name, namespace, podNamePrefix)
		klog.V(4).Infof("Memory query: %s", expr)
		if err := p.WithQuery(expr); err != nil {
			return err
		}
		mc.WithConfig(makeMemConfig(expr, a.ConfigProperties))

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

func (a *ResourceRequestAdvisor) Name() string {
	return "ResourceRequestAdvisor"
}
