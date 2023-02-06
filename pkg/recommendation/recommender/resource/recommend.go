package resource

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	recommendermodel "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/recommender/model"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/oom"
	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/recommend/types"
	"github.com/gocrane/crane/pkg/recommendation/framework"
	"github.com/gocrane/crane/pkg/utils"
)

const callerFormat = "ResourceRecommendationCaller-%s-%s"

type PatchResource struct {
	Spec PatchResourceSpec `json:"spec,omitempty"`
}

type PatchResourceSpec struct {
	Template PatchResourcePodTemplateSpec `json:"template"`
}

type PatchResourcePodTemplateSpec struct {
	Spec PatchResourcePodSpec `json:"spec,omitempty"`
}

type PatchResourcePodSpec struct {
	// +patchMergeKey=name
	// +patchStrategy=merge
	Containers []corev1.Container `json:"containers" patchStrategy:"merge" patchMergeKey:"name"`
}

func (rr *ResourceRecommender) PreRecommend(ctx *framework.RecommendationContext) error {
	return nil
}

func (rr *ResourceRecommender) makeCpuConfig() *config.Config {
	return &config.Config{
		Percentile: &predictionapi.Percentile{
			Aggregated:        true,
			HistoryLength:     rr.CpuModelHistoryLength,
			SampleInterval:    rr.CpuSampleInterval,
			MarginFraction:    rr.CpuRequestMarginFraction,
			TargetUtilization: rr.CpuTargetUtilization,
			Percentile:        rr.CpuRequestPercentile,
			Histogram: predictionapi.HistogramConfig{
				HalfLife:   "24h",
				BucketSize: "0.1",
				MaxValue:   "100",
			},
		},
	}
}

func (rr *ResourceRecommender) makeMemConfig() *config.Config {
	return &config.Config{
		Percentile: &predictionapi.Percentile{
			Aggregated:        true,
			HistoryLength:     rr.MemHistoryLength,
			SampleInterval:    rr.MemSampleInterval,
			MarginFraction:    rr.MemMarginFraction,
			Percentile:        rr.MemPercentile,
			TargetUtilization: rr.MemTargetUtilization,
			Histogram: predictionapi.HistogramConfig{
				HalfLife:   "48h",
				BucketSize: "104857600",
				MaxValue:   "104857600000",
			},
		},
	}
}

func (rr *ResourceRecommender) Recommend(ctx *framework.RecommendationContext) error {
	predictor := ctx.PredictorMgr.GetPredictor(predictionapi.AlgorithmTypePercentile)
	if predictor == nil {
		return fmt.Errorf("predictor %v not found", predictionapi.AlgorithmTypePercentile)
	}

	resourceRecommendation := &types.ResourceRequestRecommendation{}

	var newContainers []corev1.Container
	var oldContainers []corev1.Container

	oomRecords, err := rr.oomRecorder.GetOOMRecord()
	if err != nil {
		return err
	}

	namespace := ctx.Object.GetNamespace()
	for _, c := range ctx.Pods[0].Spec.Containers {
		cr := types.ContainerRecommendation{
			ContainerName: c.Name,
			Target:        map[corev1.ResourceName]string{},
		}

		caller := fmt.Sprintf(callerFormat, klog.KObj(ctx.Recommendation), ctx.Recommendation.UID)
		metricNamer := metricnaming.ResourceToContainerMetricNamer(namespace, ctx.Recommendation.Spec.TargetRef.APIVersion,
			ctx.Recommendation.Spec.TargetRef.Kind, ctx.Recommendation.Spec.TargetRef.Name, c.Name, corev1.ResourceCPU, caller)
		klog.Infof("%s: CPU query for resource request recommendation: %s", ctx.String(), metricNamer.BuildUniqueKey())
		cpuConfig := rr.makeCpuConfig()
		tsList, err := utils.QueryPredictedValuesOnce(ctx.Recommendation, predictor, caller, cpuConfig, metricNamer)
		if err != nil {
			return err
		}
		if len(tsList) < 1 || len(tsList[0].Samples) < 1 {
			return fmt.Errorf("no value retured for queryExpr: %s", metricNamer.BuildUniqueKey())
		}
		v := int64(tsList[0].Samples[0].Value * 1000)
		cpuQuantity := resource.NewMilliQuantity(v, resource.DecimalSI)
		klog.Infof("%s: container %s recommended cpu %s", ctx.String(), c.Name, cpuQuantity.String())

		metricNamer = metricnaming.ResourceToContainerMetricNamer(namespace, ctx.Recommendation.Spec.TargetRef.APIVersion,
			ctx.Recommendation.Spec.TargetRef.Kind, ctx.Recommendation.Spec.TargetRef.Name, c.Name, corev1.ResourceMemory, caller)
		klog.Infof("%s Memory query for resource request recommendation: %s", ctx.String(), metricNamer.BuildUniqueKey())
		memConfig := rr.makeMemConfig()
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
		klog.Infof("%s: container %s recommended memory %s", ctx.String(), c.Name, memQuantity.String())

		// Use oom protected memory if exist
		if rr.OOMProtection {
			oomProtectMem := rr.MemoryOOMProtection(oomRecords, namespace, ctx.Object.GetName(), c.Name)
			if oomProtectMem != nil && !oomProtectMem.IsZero() && oomProtectMem.Cmp(*memQuantity) > 0 {
				klog.Infof("%s: container %s using oomProtect Memory %s", ctx.String(), c.Name, oomProtectMem.String())
				memQuantity = oomProtectMem
			}
		}

		// Resource Specification enabled
		if rr.Specification {
			normalizedCpu, normalizedMem := GetNormalizedResource(cpuQuantity, memQuantity, rr.SpecificationConfigs)
			klog.Infof("GetNormalizedResource currentCpu %s normalizedCpu %s currentMem %s normalizedMem %s", cpuQuantity.String(), normalizedCpu.String(), memQuantity.String(), normalizedMem.String())
			if normalizedCpu.Value() > 0 && normalizedMem.Value() > 0 {
				cpuQuantity = &normalizedCpu
				memQuantity = &normalizedMem
			}
		}

		cr.Target[corev1.ResourceCPU] = cpuQuantity.String()
		cr.Target[corev1.ResourceMemory] = memQuantity.String()

		newContainerSpec := corev1.Container{
			Name: c.Name,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *cpuQuantity,
					corev1.ResourceMemory: *memQuantity,
				},
			},
		}

		oldContainerSpec := corev1.Container{
			Name: c.Name,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    c.Resources.Requests[corev1.ResourceCPU],
					corev1.ResourceMemory: c.Resources.Requests[corev1.ResourceMemory],
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    c.Resources.Limits[corev1.ResourceCPU],
					corev1.ResourceMemory: c.Resources.Limits[corev1.ResourceMemory],
				},
			},
		}

		newContainers = append(newContainers, newContainerSpec)
		oldContainers = append(oldContainers, oldContainerSpec)

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

	var newPatch PatchResource
	newPatch.Spec.Template.Spec.Containers = newContainers
	newPatchBytes, err := json.Marshal(newPatch)
	if err != nil {
		return fmt.Errorf("marshal newPatch failed %s. ", err)
	}

	var oldPatch PatchResource
	oldPatch.Spec.Template.Spec.Containers = oldContainers
	oldPatchBytes, err := json.Marshal(oldPatch)
	if err != nil {
		return fmt.Errorf("marshal oldPatch failed %s. ", err)
	}

	if reflect.DeepEqual(&newPatch, &oldPatch) {
		ctx.Recommendation.Status.Action = "None"
	} else {
		ctx.Recommendation.Status.Action = "Patch"
	}

	ctx.Recommendation.Status.RecommendedInfo = string(newPatchBytes)
	ctx.Recommendation.Status.CurrentInfo = string(oldPatchBytes)

	return nil
}

// Policy add some logic for result of recommend phase.
func (rr *ResourceRecommender) Policy(ctx *framework.RecommendationContext) error {
	return nil
}

func (rr *ResourceRecommender) MemoryOOMProtection(oomRecords []oom.OOMRecord, namespace string, workloadName string, containerName string) *resource.Quantity {
	var oomRecord *oom.OOMRecord
	for _, record := range oomRecords {
		// use oomRecord for all pods in workload
		if strings.HasPrefix(record.Pod, workloadName) && containerName == record.Container && namespace == record.Namespace {
			oomRecord = &record
			break
		}
	}

	// ignore too old oom events
	if oomRecord != nil && time.Since(oomRecord.OOMAt) <= (time.Hour*24*7) {
		memoryOOM := oomRecord.Memory.Value()
		var memoryNeeded recommendermodel.ResourceAmount

		memoryNeeded = recommendermodel.ResourceAmountMax(recommendermodel.ResourceAmount(memoryOOM)+recommendermodel.MemoryAmountFromBytes(recommendermodel.OOMMinBumpUp),
			recommendermodel.ScaleResource(recommendermodel.ResourceAmount(memoryOOM), rr.OOMBumpRatio))

		return resource.NewQuantity(int64(memoryNeeded), resource.BinarySI)
	}

	return nil
}
