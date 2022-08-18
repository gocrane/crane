package advisor

import (
	"fmt"

	"github.com/gocrane/crane/pkg/metricnaming"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/metrics"
	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/recommend/types"
	"github.com/gocrane/crane/pkg/utils"
)

const callerFormat = "RecommendationCaller-%s-%s"

const (
	DefaultNamespace = "default"
)

type ResourceRequestAdvisor struct {
	*types.Context
}

func makeCpuConfig(props map[string]string) *config.Config {
	sampleInterval, exists := props["resource.cpu-sample-interval"]
	if !exists {
		sampleInterval = "1m"
	}
	percentile, exists := props["resource.cpu-request-percentile"]
	if !exists {
		percentile = "0.99"
	}
	marginFraction, exists := props["resource.cpu-request-margin-fraction"]
	if !exists {
		marginFraction = "0.15"
	}
	targetUtilization, exists := props["resource.cpu-target-utilization"]
	if !exists {
		targetUtilization = "1.0"
	}
	historyLength, exists := props["resource.cpu-model-history-length"]
	if !exists {
		historyLength = "168h"
	}

	return &config.Config{
		Percentile: &predictionapi.Percentile{
			Aggregated:        true,
			HistoryLength:     historyLength,
			SampleInterval:    sampleInterval,
			MarginFraction:    marginFraction,
			TargetUtilization: targetUtilization,
			Percentile:        percentile,
			Histogram: predictionapi.HistogramConfig{
				HalfLife:   "24h",
				BucketSize: "0.1",
				MaxValue:   "100",
			},
		},
	}
}

func makeMemConfig(props map[string]string) *config.Config {
	sampleInterval, exists := props["resource.mem-sample-interval"]
	if !exists {
		sampleInterval = "1m"
	}
	percentile, exists := props["resource.mem-request-percentile"]
	if !exists {
		percentile = "0.99"
	}
	marginFraction, exists := props["resource.mem-request-margin-fraction"]
	if !exists {
		marginFraction = "0.15"
	}
	targetUtilization, exists := props["resource.mem-target-utilization"]
	if !exists {
		targetUtilization = "1.0"
	}
	historyLength, exists := props["resource.mem-model-history-length"]
	if !exists {
		historyLength = "168h"
	}

	return &config.Config{
		Percentile: &predictionapi.Percentile{
			Aggregated:        true,
			HistoryLength:     historyLength,
			SampleInterval:    sampleInterval,
			MarginFraction:    marginFraction,
			Percentile:        percentile,
			TargetUtilization: targetUtilization,
			Histogram: predictionapi.HistogramConfig{
				HalfLife:   "48h",
				BucketSize: "104857600",
				MaxValue:   "104857600000",
			},
		},
	}
}

func (a *ResourceRequestAdvisor) Advise(proposed *types.ProposedRecommendation) error {
	r := &types.ResourceRequestRecommendation{}

	p := a.PredictorMgr.GetPredictor(predictionapi.AlgorithmTypePercentile)
	if p == nil {
		return fmt.Errorf("predictor %v not found", predictionapi.AlgorithmTypePercentile)
	}

	if len(a.Pods) == 0 {
		return fmt.Errorf("pod not found")
	}

	pod := a.Pods[0]
	namespace := pod.Namespace

	for _, c := range pod.Spec.Containers {
		cr := types.ContainerRecommendation{
			ContainerName: c.Name,
			Target:        map[corev1.ResourceName]string{},
		}

		caller := fmt.Sprintf(callerFormat, klog.KObj(a.Recommendation), a.Recommendation.UID)
		metricNamer := metricnaming.ResourceToContainerMetricNamer(namespace, a.Recommendation.Spec.TargetRef.APIVersion,
			a.Recommendation.Spec.TargetRef.Kind, a.Recommendation.Spec.TargetRef.Name, c.Name, corev1.ResourceCPU, caller)
		klog.V(6).Infof("CPU query for resource request recommendation: %s", metricNamer.BuildUniqueKey())
		cpuConfig := makeCpuConfig(a.ConfigProperties)
		tsList, err := utils.QueryPredictedValuesOnce(a.Recommendation, p, caller, cpuConfig, metricNamer)
		if err != nil {
			return err
		}
		if len(tsList) < 1 || len(tsList[0].Samples) < 1 {
			return fmt.Errorf("no value retured for queryExpr: %s", metricNamer.BuildUniqueKey())
		}
		v := int64(tsList[0].Samples[0].Value * 1000)
		q := resource.NewMilliQuantity(v, resource.DecimalSI)
		cr.Target[corev1.ResourceCPU] = q.String()
		// export recommended values as prom metrics
		a.recordResourceRecommendation(c.Name, corev1.ResourceCPU, q)

		metricNamer = metricnaming.ResourceToContainerMetricNamer(namespace, a.Recommendation.Spec.TargetRef.APIVersion,
			a.Recommendation.Spec.TargetRef.Kind, a.Recommendation.Spec.TargetRef.Name, c.Name, corev1.ResourceMemory, caller)
		klog.V(6).Infof("Memory query for resource request recommendation: %s", metricNamer.BuildUniqueKey())
		memConfig := makeMemConfig(a.ConfigProperties)
		tsList, err = utils.QueryPredictedValuesOnce(a.Recommendation, p, caller, memConfig, metricNamer)
		if err != nil {
			return err
		}
		if len(tsList) < 1 || len(tsList[0].Samples) < 1 {
			return fmt.Errorf("no value retured for queryExpr: %s", metricNamer.BuildUniqueKey())
		}
		v = int64(tsList[0].Samples[0].Value)
		if v <= 0 {
			return fmt.Errorf("no enough metrics")
		}
		q = resource.NewQuantity(v, resource.BinarySI)
		cr.Target[corev1.ResourceMemory] = q.String()
		// export recommended values as prom metrics
		a.recordResourceRecommendation(c.Name, corev1.ResourceMemory, q)

		r.Containers = append(r.Containers, cr)
	}

	proposed.ResourceRequest = r
	return nil
}

func (a *ResourceRequestAdvisor) recordResourceRecommendation(containerName string, resName corev1.ResourceName, quantity *resource.Quantity) {
	labels := map[string]string{
		"apiversion": a.Recommendation.Spec.TargetRef.APIVersion,
		"owner_kind": a.Recommendation.Spec.TargetRef.Kind,
		"namespace":  a.Recommendation.Spec.TargetRef.Namespace,
		"owner_name": a.Recommendation.Spec.TargetRef.Name,
		"container":  containerName,
		"resource":   resName.String(),
	}

	// record owner replicas
	if a.Scale != nil {
		labels["owner_replicas"] = fmt.Sprintf("%d", a.Scale.Spec.Replicas)
	} else if a.DaemonSet != nil {
		labels["owner_replicas"] = fmt.Sprintf("%d", a.DaemonSet.Status.NumberAvailable)
	}

	switch resName {
	case corev1.ResourceCPU:
		metrics.ResourceRecommendation.With(labels).Set(float64(quantity.MilliValue()) / 1000.)
	case corev1.ResourceMemory:
		metrics.ResourceRecommendation.With(labels).Set(float64(quantity.Value()))
	}
}

func (a *ResourceRequestAdvisor) Name() string {
	return "ResourceRequestAdvisor"
}
