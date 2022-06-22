package utils

import (
	"fmt"

	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/api/core/v1"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"

	"github.com/gocrane/crane/pkg/known"
)

func IsEHPAPredictionEnabled(ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler) bool {
	return ehpa.Spec.Prediction != nil && ehpa.Spec.Prediction.PredictionWindowSeconds != nil && ehpa.Spec.Prediction.PredictionAlgorithm != nil
}

func IsEHPAHasPredictionMetric(ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler) bool {
	for _, metric := range ehpa.Spec.Metrics {
		if metric.Type == autoscalingv2.ResourceMetricSourceType {
			metricName := GetPredictionMetricName(metric.Resource.Name)
			if len(metricName) == 0 {
				continue
			}
			return true
		}
	}
	return false
}

func IsEHPACronEnabled(ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler) bool {
	return len(ehpa.Spec.Crons) > 0
}

// GetPredictionMetricName return metric name used by prediction
func GetPredictionMetricName(name v1.ResourceName) string {
	switch name {
	case v1.ResourceCPU:
		return known.MetricNamePodCpuUsage
	default:
		return ""
	}
}

// GetExternalPredictionMetricName return metric name used by prediction
func GetExternalPredictionMetricName(name string) string {
	return fmt.Sprintf("crane-%s", name)
}
