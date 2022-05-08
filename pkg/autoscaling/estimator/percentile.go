package estimator

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/metricquery"
	"github.com/gocrane/crane/pkg/prediction"
	predictionconfig "github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/utils/target"
)

const callerFormat = "EVPACaller-%s-%s"

type PercentileResourceEstimator struct {
	Predictor     prediction.Interface
	Client        client.Client
	TargetFetcher target.SelectorFetcher
}

func (e *PercentileResourceEstimator) GetResourceEstimation(evpa *autoscalingapi.EffectiveVerticalPodAutoscaler, config map[string]string, containerName string, currRes *corev1.ResourceRequirements) (corev1.ResourceList, error) {
	recommendResource := corev1.ResourceList{}

	selector, err := e.TargetFetcher.Fetch(&corev1.ObjectReference{
		APIVersion: evpa.Spec.TargetRef.APIVersion,
		Kind:       evpa.Spec.TargetRef.Kind,
		Name:       evpa.Spec.TargetRef.Name,
		Namespace:  evpa.Namespace,
	})
	if err != nil {
		klog.ErrorS(err, "Failed to fetch evpa target workload selector.", "evpa", klog.KObj(evpa))
	}
	caller := fmt.Sprintf(callerFormat, klog.KObj(evpa), string(evpa.UID))
	cpuMetricNamer := &metricnaming.GeneralMetricNamer{
		CallerName: caller,
		Metric: &metricquery.Metric{
			Type:       metricquery.ContainerMetricType,
			MetricName: corev1.ResourceCPU.String(),
			Container: &metricquery.ContainerNamerInfo{
				Namespace:     evpa.Namespace,
				WorkloadName:  evpa.Spec.TargetRef.Name,
				ContainerName: containerName,
				Selector:      selector,
			},
		},
	}

	cpuConfig := getCpuConfig(config)

	memoryMetricNamer := &metricnaming.GeneralMetricNamer{
		CallerName: caller,
		Metric: &metricquery.Metric{
			Type:       metricquery.ContainerMetricType,
			MetricName: corev1.ResourceMemory.String(),
			Container: &metricquery.ContainerNamerInfo{
				Namespace:     evpa.Namespace,
				WorkloadName:  evpa.Spec.TargetRef.Name,
				ContainerName: containerName,
				Selector:      selector,
			},
		},
	}
	memConfig := getMemConfig(config)

	var errs []error
	// first register cpu & memory, or the memory will be not registered before the cpu prediction succeed
	err1 := e.Predictor.WithQuery(cpuMetricNamer, caller, *cpuConfig)
	if err1 != nil {
		errs = append(errs, err1)
	}
	err2 := e.Predictor.WithQuery(memoryMetricNamer, caller, *memConfig)
	if err2 != nil {
		errs = append(errs, err2)
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to register metricNamer: %v", errs)
	}

	var predictErrs []error
	var noValueErrs []error
	tsList, err := e.Predictor.QueryRealtimePredictedValues(context.TODO(), cpuMetricNamer)
	if err != nil {
		predictErrs = append(predictErrs, err)
	}

	if len(tsList) > 0 && len(tsList[0].Samples) > 0 {
		cpuValue := int64(tsList[0].Samples[0].Value * 1000)
		recommendResource[corev1.ResourceCPU] = *resource.NewMilliQuantity(cpuValue, resource.DecimalSI)
	} else {
		noValueErrs = append(noValueErrs, fmt.Errorf("no value retured for queryExpr: %s", cpuMetricNamer.BuildUniqueKey()))
	}

	tsList, err = e.Predictor.QueryRealtimePredictedValues(context.TODO(), memoryMetricNamer)
	if err != nil {
		predictErrs = append(predictErrs, err)
	}

	if len(tsList) > 0 && len(tsList[0].Samples) > 0 {
		memValue := int64(tsList[0].Samples[0].Value)
		recommendResource[corev1.ResourceMemory] = *resource.NewQuantity(memValue, resource.BinarySI)
	} else {
		noValueErrs = append(noValueErrs, fmt.Errorf("no value retured for queryExpr: %s", memoryMetricNamer.BuildUniqueKey()))
	}

	// all failed
	if len(recommendResource) == 0 {
		return recommendResource, fmt.Errorf("all resource predicted failed, predictErrs: %v, noValueErrs: %v", predictErrs, noValueErrs)
	}

	// at least one succeed
	return recommendResource, nil
}

func (e *PercentileResourceEstimator) DeleteEstimation(evpa *autoscalingapi.EffectiveVerticalPodAutoscaler) {
	selector, err := e.TargetFetcher.Fetch(&corev1.ObjectReference{
		APIVersion: evpa.Spec.TargetRef.APIVersion,
		Kind:       evpa.Spec.TargetRef.Kind,
		Name:       evpa.Spec.TargetRef.Name,
		Namespace:  evpa.Namespace,
	})
	if err != nil {
		klog.ErrorS(err, "Failed to fetch evpa target workload selector.", "evpa", klog.KObj(evpa))
	}
	for _, containerPolicy := range evpa.Spec.ResourcePolicy.ContainerPolicies {
		caller := fmt.Sprintf(callerFormat, klog.KObj(evpa), string(evpa.UID))
		cpuMetricNamer := &metricnaming.GeneralMetricNamer{
			CallerName: caller,
			Metric: &metricquery.Metric{
				Type:       metricquery.ContainerMetricType,
				MetricName: corev1.ResourceCPU.String(),
				Container: &metricquery.ContainerNamerInfo{
					Namespace:     evpa.Namespace,
					WorkloadName:  evpa.Spec.TargetRef.Name,
					ContainerName: containerPolicy.ContainerName,
					Selector:      selector,
				},
			},
		}
		err := e.Predictor.DeleteQuery(cpuMetricNamer, caller)
		if err != nil {
			klog.ErrorS(err, "Failed to delete query.", "queryExpr", cpuMetricNamer.BuildUniqueKey())
		}
		memoryMetricNamer := &metricnaming.GeneralMetricNamer{
			CallerName: caller,
			Metric: &metricquery.Metric{
				Type:       metricquery.ContainerMetricType,
				MetricName: corev1.ResourceMemory.String(),
				Container: &metricquery.ContainerNamerInfo{
					Namespace:     evpa.Namespace,
					WorkloadName:  evpa.Spec.TargetRef.Name,
					ContainerName: containerPolicy.ContainerName,
					Selector:      selector,
				},
			},
		}
		err = e.Predictor.DeleteQuery(memoryMetricNamer, caller)
		if err != nil {
			klog.ErrorS(err, "Failed to delete query.", "queryExpr", memoryMetricNamer.BuildUniqueKey())
		}
	}
	return
}

func getCpuConfig(config map[string]string) *predictionconfig.Config {
	sampleInterval, exists := config["cpu-sample-interval"]
	if !exists {
		sampleInterval = "1m"
	}
	percentile, exists := config["cpu-request-percentile"]
	if !exists {
		percentile = "0.99"
	}
	marginFraction, exists := config["cpu-request-margin-fraction"]
	if !exists {
		marginFraction = "0.15"
	}

	initModeStr, exists := config["cpu-model-init-mode"]
	initMode := predictionconfig.ModelInitModeLazyTraining
	if !exists {
		initMode = predictionconfig.ModelInitMode(initModeStr)
	}

	historyLength, exists := config["cpu-model-history-length"]
	if !exists {
		historyLength = "24h"
	}

	return &predictionconfig.Config{
		InitMode: &initMode,
		Percentile: &predictionapi.Percentile{
			Aggregated:     true,
			HistoryLength:  historyLength,
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

func getMemConfig(props map[string]string) *predictionconfig.Config {
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

	initModeStr, exists := props["mem-model-init-mode"]
	initMode := predictionconfig.ModelInitModeLazyTraining
	if !exists {
		initMode = predictionconfig.ModelInitMode(initModeStr)
	}

	historyLength, exists := props["mem-model-history-length"]
	if !exists {
		historyLength = "48h"
	}

	return &predictionconfig.Config{
		InitMode: &initMode,
		Percentile: &predictionapi.Percentile{
			Aggregated:     true,
			HistoryLength:  historyLength,
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
