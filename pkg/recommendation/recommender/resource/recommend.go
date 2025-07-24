package resource

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/errors"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	recommendermodel "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/recommender/model"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/features"
	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/oom"
	"github.com/gocrane/crane/pkg/prediction"
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
				BucketSize: rr.CpuHistogramBucketSize,
				MaxValue:   rr.CpuHistogramMaxValue,
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
				BucketSize: rr.MemHistogramBucketSize,
				MaxValue:   rr.MemHistogramMaxValue,
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
	namespace := ctx.Object.GetNamespace()
	caller := fmt.Sprintf(callerFormat, klog.KObj(ctx.Recommendation), ctx.Recommendation.UID)

	var newContainers []corev1.Container
	var oldContainers []corev1.Container

	oomRecords, err := ctx.OOMRecorder.GetOOMRecord()
	if err != nil {
		return err
	}

	// pod
	if utilfeature.DefaultFeatureGate.Enabled(features.EnablePodRecommendation) {
		cpuTsList, memoryTsList, usePodMetrics, err := rr.getPodCpuAndMemoryTsList(ctx, namespace, caller, predictor)
		if err != nil {
			klog.Warningf("getPodCpuAndMemoryTsList err: %v", err)
		}
		if usePodMetrics {
			klog.V(4).Infof("use pod metrics for pod %s", ctx.Pods[0].Name)
			pr := types.PodRecommendation{
				PodName: ctx.Pods[0].Name,
				Target:  map[corev1.ResourceName]string{},
			}

			cpuQuantity, memQuantity, err := rr.recommendCpuAndMemResources(ctx, cpuTsList, memoryTsList, oomRecords, namespace, ctx.Object.GetName(), ctx.Pods[0].Name)
			if err != nil {
				klog.Errorf("recommendCpuAndMemResources %v", err)
			}

			if cpuQuantity != nil {
				pr.Target[corev1.ResourceCPU] = cpuQuantity.String()
			}
			if memQuantity != nil {
				pr.Target[corev1.ResourceMemory] = memQuantity.String()
			}

			if len(pr.Target) != 0 {
				resourceRecommendation.Pod = &pr
			}
		} else {
			klog.V(4).Infof("not use pod metrics for pod %s", ctx.Pods[0].Name)
		}
	}

	// containers
	for _, c := range ctx.Pods[0].Spec.Containers {
		cpuTsList, memTsList, err := rr.getContainerCpuAndMemoryTsList(ctx, predictor, caller, namespace, c.Name)
		if err != nil {
			return err
		}

		cpuQuantity, memQuantity, err := rr.recommendCpuAndMemResources(ctx, cpuTsList, memTsList, oomRecords, namespace, ctx.Object.GetName(), c.Name)
		if err != nil {
			return err
		}
		if cpuQuantity == nil || memQuantity == nil {
			return fmt.Errorf("resource recommendation failed for container %s: cpu=%v, memory=%v", c.Name, cpuQuantity != nil, memQuantity != nil)
		}

		cr := types.ContainerRecommendation{
			ContainerName: c.Name,
			Target:        map[corev1.ResourceName]string{},
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

// getContainerCpuAndMemoryTsList gets container metrics data
func (rr *ResourceRecommender) getContainerCpuAndMemoryTsList(ctx *framework.RecommendationContext,
	predictor prediction.Interface,
	caller string,
	namespace, containerName string) ([]*common.TimeSeries, []*common.TimeSeries, error) {

	// cpu
	cpuNamer := metricnaming.ResourceToContainerMetricNamer(namespace,
		ctx.Recommendation.Spec.TargetRef.APIVersion,
		ctx.Recommendation.Spec.TargetRef.Kind,
		ctx.Recommendation.Spec.TargetRef.Name,
		containerName,
		corev1.ResourceCPU,
		caller)

	cpuTs, err := utils.QueryPredictedValuesOnce(ctx.Recommendation, predictor, caller, rr.makeCpuConfig(), cpuNamer)
	if err != nil {
		return nil, nil, err
	}

	// memory
	memNamer := metricnaming.ResourceToContainerMetricNamer(namespace,
		ctx.Recommendation.Spec.TargetRef.APIVersion,
		ctx.Recommendation.Spec.TargetRef.Kind,
		ctx.Recommendation.Spec.TargetRef.Name,
		containerName,
		corev1.ResourceMemory,
		caller)

	memTs, err := utils.QueryPredictedValuesOnce(ctx.Recommendation, predictor, caller, rr.makeMemConfig(), memNamer)
	if err != nil {
		return nil, nil, err
	}

	return cpuTs, memTs, nil
}

func (rr *ResourceRecommender) getPodCpuAndMemoryTsList(ctx *framework.RecommendationContext, namespace, caller string, predictor prediction.Interface) ([]*common.TimeSeries, []*common.TimeSeries, bool, error) {
	var errs []error
	cpuOK, memOK := true, true

	// cpu
	cpuMetricNamer := metricnaming.ResourceToPodMetricNamer(namespace,
		ctx.Pods[0].Name,
		corev1.ResourceCPU,
		caller)
	cpuTsList, err := utils.QueryPredictedValuesOnce(ctx.Recommendation, predictor, caller, rr.makeCpuConfig(), cpuMetricNamer)
	if err != nil {
		cpuOK = false
		errs = append(errs, err)
	}

	// memory
	memoryMetricNamer := metricnaming.ResourceToPodMetricNamer(namespace,
		ctx.Pods[0].Name,
		corev1.ResourceMemory,
		caller)
	memTsList, err := utils.QueryPredictedValuesOnce(ctx.Recommendation, predictor, caller, rr.makeMemConfig(), memoryMetricNamer)
	if err != nil {
		memOK = false
		errs = append(errs, err)
	}

	if !cpuOK && !memOK {
		return nil, nil, false, errors.NewAggregate(errs)
	}

	return cpuTsList, memTsList, true, errors.NewAggregate(errs)
}

// recommendCpuAndMemResources recommends CPU and memory resources based on historical monitoring data, OOM records, and resource specification normalization
func (rr *ResourceRecommender) recommendCpuAndMemResources(ctx *framework.RecommendationContext,
	cpuTsList []*common.TimeSeries,
	memTsList []*common.TimeSeries,
	oomRecords []oom.OOMRecord,
	namespace, workloadName, containerName string) (*resource.Quantity, *resource.Quantity, error) {

	var errs []error
	cpuOK, memOK := true, true

	// cpu
	cpuQuantity, err := rr.recommendSingleResource(ctx, cpuTsList, rr.CpuModelHistoryLength, corev1.ResourceCPU, containerName)
	if err != nil {
		cpuOK = false
		errs = append(errs, err)
	}

	// memory
	memQuantity, err := rr.recommendSingleResource(ctx, memTsList, rr.MemHistoryLength, corev1.ResourceMemory, containerName)
	if err != nil {
		memOK = false
		errs = append(errs, err)
	}

	if !cpuOK && !memOK {
		return nil, nil, errors.NewAggregate(errs)
	}

	// adjust memory recommendations by analyzing historical OOM events
	if memOK && rr.OOMProtection {
		if oomMem := rr.MemoryOOMProtection(oomRecords, namespace, workloadName, containerName); oomMem != nil {
			if !oomMem.IsZero() && oomMem.Cmp(*memQuantity) > 0 {
				klog.Infof("%s: %s using oomProtect Memory %s", ctx.String(), containerName, oomMem.String())
				memQuantity = oomMem
			}
		}
	}

	// standardize resource recommendations to predefined specifications
	if rr.Specification {
		if cpuOK && memOK {
			normalizedCpu, normalizedMem := GetNormalizedResource(cpuQuantity, memQuantity, rr.SpecificationConfigs)
			klog.Infof("GetNormalizedResource currentCpu %s normalizedCpu %s currentMem %s normalizedMem %s",
				cpuQuantity.String(), normalizedCpu.String(), memQuantity.String(), normalizedMem.String())
			if normalizedCpu.Value() > 0 && normalizedMem.Value() > 0 {
				cpuQuantity = &normalizedCpu
				memQuantity = &normalizedMem
			}
		} else {
			return nil, nil, fmt.Errorf("cpu or memory recommendation failed, cannot standardize resource recommendations to predefined specifications")
		}
	}

	return cpuQuantity, memQuantity, nil
}

func (rr *ResourceRecommender) recommendSingleResource(ctx *framework.RecommendationContext,
	tsList []*common.TimeSeries,
	historyLength string,
	resourceType corev1.ResourceName,
	containerName string) (*resource.Quantity, error) {

	if len(tsList) == 0 || len(tsList[0].Samples) == 0 {
		return nil, fmt.Errorf("no metrics data for %s", resourceType)
	}

	if rr.HistoryCompletionCheck {
		completion, existDays, err := utils.DetectTimestampCompletion(tsList, historyLength, time.Now())
		if !completion || err != nil {
			return nil, fmt.Errorf("%s timestamps not completed: expect %s actual %d days", resourceType, historyLength, existDays)
		}
	}

	value := tsList[0].Samples[0].Value
	var quantity *resource.Quantity
	if resourceType == corev1.ResourceCPU {
		value *= 1000
		quantity = resource.NewMilliQuantity(int64(value), resource.DecimalSI)
	} else if resourceType == corev1.ResourceMemory {
		quantity = resource.NewQuantity(int64(value), resource.BinarySI)
		if value <= 0 {
			return nil, fmt.Errorf("invalid %s value: %f", resourceType, value)
		}
	}

	klog.Infof("%s: %s recommended %s %s", ctx.String(), containerName, resourceType, quantity.String())
	return quantity, nil
}
