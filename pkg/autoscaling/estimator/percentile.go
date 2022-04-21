package estimator

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/metricquery"
	"github.com/gocrane/crane/pkg/prediction"
	predictionconfig "github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/utils"
)

const callerFormat = "EVPACaller-%s-%s-%s"

type PercentileResourceEstimator struct {
	Predictor prediction.Interface
	Client    client.Client
}

func (e *PercentileResourceEstimator) GetResourceEstimation(evpa *autoscalingapi.EffectiveVerticalPodAutoscaler, config map[string]string, containerName string, currRes *corev1.ResourceRequirements) (corev1.ResourceList, error) {
	recommendResource := corev1.ResourceList{}

	cpuMetricNamer := &metricnaming.GeneralMetricNamer{
		Metric: &metricquery.Metric{
			Type:       metricquery.ContainerMetricType,
			MetricName: corev1.ResourceCPU.String(),
			Container: &metricquery.ContainerNamerInfo{
				Namespace:     evpa.Namespace,
				WorkloadName:  evpa.Spec.TargetRef.Name,
				ContainerName: containerName,
				Selector:      labels.Everything(),
			},
		},
	}

	cpuConfig := getCpuConfig(config)
	tsList, err := utils.QueryPredictedValues(e.Predictor, fmt.Sprintf(callerFormat, string(evpa.UID), containerName, corev1.ResourceCPU), cpuConfig, cpuMetricNamer)
	if err != nil {
		return nil, err
	}

	if len(tsList) < 1 || len(tsList[0].Samples) < 1 {
		return nil, fmt.Errorf("no value retured for queryExpr: %s", cpuMetricNamer.BuildUniqueKey())
	}

	cpuValue := int64(tsList[0].Samples[0].Value * 1000)
	recommendResource[corev1.ResourceCPU] = *resource.NewMilliQuantity(cpuValue, resource.DecimalSI)

	memoryMetricNamer := &metricnaming.GeneralMetricNamer{
		Metric: &metricquery.Metric{
			Type:       metricquery.ContainerMetricType,
			MetricName: corev1.ResourceMemory.String(),
			Container: &metricquery.ContainerNamerInfo{
				Namespace:     evpa.Namespace,
				WorkloadName:  evpa.Spec.TargetRef.Name,
				ContainerName: containerName,
				Selector:      labels.Everything(),
			},
		},
	}

	memConfig := getMemConfig(config)
	tsList, err = utils.QueryPredictedValues(e.Predictor, fmt.Sprintf(callerFormat, string(evpa.UID), containerName, corev1.ResourceMemory), memConfig, memoryMetricNamer)
	if err != nil {
		return nil, err
	}

	if len(tsList) < 1 || len(tsList[0].Samples) < 1 {
		return nil, fmt.Errorf("no value retured for queryExpr: %s", memoryMetricNamer.BuildUniqueKey())
	}

	memValue := int64(tsList[0].Samples[0].Value)
	recommendResource[corev1.ResourceMemory] = *resource.NewQuantity(memValue, resource.BinarySI)

	return recommendResource, nil
}

func (e *PercentileResourceEstimator) DeleteEstimation(evpa *autoscalingapi.EffectiveVerticalPodAutoscaler) {
	for _, containerPolicy := range evpa.Spec.ResourcePolicy.ContainerPolicies {
		cpuMetricNamer := &metricnaming.GeneralMetricNamer{
			Metric: &metricquery.Metric{
				Type:       metricquery.ContainerMetricType,
				MetricName: corev1.ResourceCPU.String(),
				Container: &metricquery.ContainerNamerInfo{
					Namespace:     evpa.Namespace,
					WorkloadName:  evpa.Spec.TargetRef.Name,
					ContainerName: containerPolicy.ContainerName,
					Selector:      labels.Everything(),
				},
			},
		}
		err := e.Predictor.DeleteQuery(cpuMetricNamer, fmt.Sprintf(callerFormat, string(evpa.UID), containerPolicy.ContainerName, corev1.ResourceCPU))
		if err != nil {
			klog.ErrorS(err, "Failed to delete query.", "queryExpr", cpuMetricNamer.BuildUniqueKey())
		}

		memoryMetricNamer := &metricnaming.GeneralMetricNamer{
			Metric: &metricquery.Metric{
				Type:       metricquery.ContainerMetricType,
				MetricName: corev1.ResourceMemory.String(),
				Container: &metricquery.ContainerNamerInfo{
					Namespace:     evpa.Namespace,
					WorkloadName:  evpa.Spec.TargetRef.Name,
					ContainerName: containerPolicy.ContainerName,
					Selector:      labels.Everything(),
				},
			},
		}
		err = e.Predictor.DeleteQuery(memoryMetricNamer, fmt.Sprintf(callerFormat, string(evpa.UID), containerPolicy.ContainerName, corev1.ResourceMemory))
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

	return &predictionconfig.Config{
		Percentile: &predictionapi.Percentile{
			Aggregated:     true,
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

	return &predictionconfig.Config{
		Percentile: &predictionapi.Percentile{
			Aggregated:     true,
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
