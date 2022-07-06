package utils

import (
	"fmt"
	"regexp"
	"strings"

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

	for key := range ehpa.Annotations {
		if strings.HasPrefix(key, known.EffectiveHorizontalPodAutoscalerExternalMetricsAnnotationPrefix) {
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

// GetGeneralPredictionMetricName return metric name used by prediction
func GetGeneralPredictionMetricName(sourceType autoscalingv2.MetricSourceType, isCron bool, name string) string {
	prefix := ""

	switch sourceType {
	case autoscalingv2.PodsMetricSourceType:
		prefix = "custom.pods"
	case autoscalingv2.ExternalMetricSourceType:
		prefix = "external"
	}

	if isCron {
		prefix = "cron"
	}

	return fmt.Sprintf("crane_%s_%s", prefix, name)
}

// GetExpressionQuery return metric query from annotation by metricName
func GetExpressionQuery(metricName string, annotations map[string]string) string {
	for k, v := range annotations {
		if strings.HasPrefix(k, known.EffectiveHorizontalPodAutoscalerExternalMetricsAnnotationPrefix) {
			compileRegex := regexp.MustCompile(fmt.Sprintf("%s(.*)", known.EffectiveHorizontalPodAutoscalerExternalMetricsAnnotationPrefix))
			matchArr := compileRegex.FindStringSubmatch(k)
			if len(matchArr) == 2 && matchArr[1][1:] == metricName {
				return v
			}
		}
	}

	return ""
}
