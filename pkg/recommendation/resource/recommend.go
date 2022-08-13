package resource

import (
	"encoding/json"
	"fmt"
	jsonpatch "github.com/evanphx/json-patch"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/recommend/types"
	"github.com/gocrane/crane/pkg/recommendation/framework"
	"github.com/gocrane/crane/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

const callerFormat = "RecommendationCaller-%s-%s"

func (rr *ResourceRecommender) PreRecommend(ctx *framework.RecommendationContext) error {
	return nil
}

func makeCpuConfig(props map[string]string) *config.Config {
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
	targetUtilization, exists := props["cpu-target-utilization"]
	if !exists {
		targetUtilization = "1.0"
	}
	historyLength, exists := props["cpu-model-history-length"]
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
	targetUtilization, exists := props["mem-target-utilization"]
	if !exists {
		targetUtilization = "1.0"
	}
	historyLength, exists := props["mem-model-history-length"]
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

func (rr *ResourceRecommender) Recommend(ctx *framework.RecommendationContext) error {
	if len(ctx.Pods) == 0 {
		return fmt.Errorf("pod not found")
	}

	predictor := ctx.PredictorMgr.GetPredictor(predictionapi.AlgorithmTypePercentile)
	if predictor == nil {
		return fmt.Errorf("predictor %v not found", predictionapi.AlgorithmTypePercentile)
	}

	resourceRecommendation := &types.ResourceRequestRecommendation{}
	newPodTemplate := ctx.PodTemplate.DeepCopy()

	namespace := ctx.Object.GetNamespace()
	for _, c := range ctx.PodTemplate.Spec.Containers {
		cr := types.ContainerRecommendation{
			ContainerName: c.Name,
			Target:        map[corev1.ResourceName]string{},
		}

		caller := fmt.Sprintf(callerFormat, klog.KObj(ctx.Recommendation), ctx.Recommendation.UID)
		metricNamer := metricnaming.ResourceToContainerMetricNamer(namespace, ctx.Recommendation.Spec.TargetRef.APIVersion,
			ctx.Recommendation.Spec.TargetRef.Kind, ctx.Recommendation.Spec.TargetRef.Name, c.Name, corev1.ResourceCPU, caller)
		klog.V(6).Infof("CPU query for resource request recommendation: %s", metricNamer.BuildUniqueKey())
		cpuConfig := makeCpuConfig(rr.Config)
		tsList, err := utils.QueryPredictedValuesOnce(ctx.Recommendation, predictor, caller, cpuConfig, metricNamer)
		if err != nil {
			return err
		}
		if len(tsList) < 1 || len(tsList[0].Samples) < 1 {
			return fmt.Errorf("no value retured for queryExpr: %s", metricNamer.BuildUniqueKey())
		}
		v := int64(tsList[0].Samples[0].Value * 1000)
		cpuQuantity := resource.NewMilliQuantity(v, resource.DecimalSI)
		cr.Target[corev1.ResourceCPU] = cpuQuantity.String()

		metricNamer = metricnaming.ResourceToContainerMetricNamer(namespace, ctx.Recommendation.Spec.TargetRef.APIVersion,
			ctx.Recommendation.Spec.TargetRef.Kind, ctx.Recommendation.Spec.TargetRef.Name, c.Name, corev1.ResourceMemory, caller)
		klog.V(6).Infof("Memory query for resource request recommendation: %s", metricNamer.BuildUniqueKey())
		memConfig := makeMemConfig(rr.Config)
		tsList, err = utils.QueryPredictedValuesOnce(ctx.Recommendation, predictor, caller, memConfig, metricNamer)
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
		memQuantity := resource.NewQuantity(v, resource.BinarySI)
		cr.Target[corev1.ResourceMemory] = memQuantity.String()

		for index := range newPodTemplate.Spec.Containers {
			if newPodTemplate.Spec.Containers[index].Name == c.Name {
				newPodTemplate.Spec.Containers[index].Resources.Requests[corev1.ResourceCPU] = *cpuQuantity
				newPodTemplate.Spec.Containers[index].Resources.Requests[corev1.ResourceMemory] = *cpuQuantity
			}
		}

		resourceRecommendation.Containers = append(resourceRecommendation.Containers, cr)
	}

	value := types.ProposedRecommendation{
		ResourceRequest: resourceRecommendation,
	}

	valueBytes, err := yaml.Marshal(value)
	if err != nil {
		return fmt.Errorf("%s yaml marshal failed: %v", rr.Name(), err)
	}

	ctx.Recommendation.Status.RecommendedValue = string(valueBytes)

	newPodTemplateBytes, err := json.Marshal(newPodTemplate)
	if err != nil {
		return err
	}

	newPodTemplateFields := map[string]interface{}{}
	if err := json.Unmarshal(newPodTemplateBytes, newPodTemplateFields); err != nil {
		return err
	}

	unstructObject := ctx.Object.(*unstructured.Unstructured)
	newUnstructObject := unstructObject.DeepCopyObject().(*unstructured.Unstructured)
	unstructured.SetNestedField(newUnstructObject.Object, newPodTemplateFields, "spec", "template", "spec")

	oldBytes, err := json.Marshal(unstructObject)
	if err != nil {
		return fmt.Errorf("encode error %s. ", err)
	}

	newBytes, err := json.Marshal(newUnstructObject)
	if err != nil {
		return fmt.Errorf("encode error %s. ", err)
	}

	newPatch, err := jsonpatch.CreateMergePatch(newBytes, oldBytes)
	if err != nil {
		return fmt.Errorf("create merge patch error %s. ", err)
	}
	oldPatch, err := jsonpatch.CreateMergePatch(oldBytes, newBytes)
	if err != nil {
		return fmt.Errorf("create merge patch error %s. ", err)
	}

	ctx.Recommendation.Status.RecommendedInfo = string(newPatch)
	ctx.Recommendation.Status.CurrentInfo = string(oldPatch)
	// TODO(qmhu) Create action type.
	ctx.Recommendation.Status.Action = "Patch"

	return nil
}

// Policy add some logic for result of recommend phase.
func (rr *ResourceRecommender) Policy(ctx *framework.RecommendationContext) error {
	return nil
}
