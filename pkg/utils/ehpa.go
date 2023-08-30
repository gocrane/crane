package utils

import (
	"fmt"
	"regexp"
	"strings"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	"github.com/gocrane/crane/pkg/known"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
)

func IsEHPAPredictionEnabled(ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler) bool {
	return ehpa.Spec.Prediction != nil && ehpa.Spec.Prediction.PredictionWindowSeconds != nil && ehpa.Spec.Prediction.PredictionAlgorithm != nil
}

func IsEHPAHasPredictionMetric(ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler) bool {
	for _, metric := range ehpa.Spec.Metrics {
		metricName := GetPredictionMetricName(metric.Type)
		if len(metricName) == 0 {
			continue
		}
		return true
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
func GetPredictionMetricName(sourceType autoscalingv2.MetricSourceType) (metricName string) {
	switch sourceType {
	case autoscalingv2.ResourceMetricSourceType, autoscalingv2.ContainerResourceMetricSourceType, autoscalingv2.PodsMetricSourceType, autoscalingv2.ExternalMetricSourceType:
		metricName = known.MetricNamePrediction
	}

	return metricName
}

// GetCronMetricName return metric name used by cron
func GetCronMetricName() string {
	return known.MetricNameCron
}

func GetMetricName(metric autoscalingv2.MetricSpec) string {
	switch metric.Type {
	case autoscalingv2.PodsMetricSourceType:
		return metric.Pods.Metric.Name
	case autoscalingv2.ResourceMetricSourceType:
		return metric.Resource.Name.String()
	case autoscalingv2.ContainerResourceMetricSourceType:
		return metric.ContainerResource.Name.String()
	case autoscalingv2.ExternalMetricSourceType:
		return metric.External.Metric.Name
	default:
		return ""
	}
}

// GetPredictionMetricIdentifier return metric name used by prediction
func GetPredictionMetricIdentifier(metric autoscalingv2.MetricSpec) string {
	var prefix string
	switch metric.Type {
	case autoscalingv2.PodsMetricSourceType:
		prefix = "pods"
	case autoscalingv2.ResourceMetricSourceType:
		prefix = "resource"
	case autoscalingv2.ContainerResourceMetricSourceType:
		prefix = "container-resource"
	case autoscalingv2.ExternalMetricSourceType:
		prefix = "external"
	}

	return fmt.Sprintf("%s.%s", prefix, GetMetricName(metric))
}

// GetExpressionQueryAnnotation return metric query from annotation by metricName
func GetExpressionQueryAnnotation(metricIdentifier string, annotations map[string]string) string {
	for k, v := range annotations {
		if strings.HasPrefix(k, known.EffectiveHorizontalPodAutoscalerExternalMetricsAnnotationPrefix) {
			compileRegex := regexp.MustCompile(fmt.Sprintf("%s(.*)", known.EffectiveHorizontalPodAutoscalerExternalMetricsAnnotationPrefix))
			matchArr := compileRegex.FindStringSubmatch(k)
			if len(matchArr) == 2 && matchArr[1][1:] == metricIdentifier {
				return v
			}
		}
	}

	return ""
}

func IsExpressionQueryAnnotationEnabled(metricIdentifier string, annotations map[string]string) bool {
	for k := range annotations {
		if strings.HasPrefix(k, known.EffectiveHorizontalPodAutoscalerExternalMetricsAnnotationPrefix) {
			compileRegex := regexp.MustCompile(fmt.Sprintf("%s(.*)", known.EffectiveHorizontalPodAutoscalerExternalMetricsAnnotationPrefix))
			matchArr := compileRegex.FindStringSubmatch(k)
			if len(matchArr) == 2 && matchArr[1][1:] == metricIdentifier {
				return true
			}
		}
	}

	return false
}

// GetExpressionQueryDefault return default metric query
func GetExpressionQueryDefault(metric autoscalingv2.MetricSpec, namespace string, name string, kind string) string {
	var expressionQuery string
	switch metric.Type {
	case autoscalingv2.ResourceMetricSourceType:
		switch metric.Resource.Name {
		case "cpu":
			expressionQuery = GetWorkloadCpuUsageExpression(namespace, name, kind)
		case "memory":
			expressionQuery = GetWorkloadMemUsageExpression(namespace, name, kind)
		}
	case autoscalingv2.ContainerResourceMetricSourceType:
		switch metric.ContainerResource.Name {
		case "cpu":
			expressionQuery = GetContainerCpuUsageExpression(namespace, name, kind, metric.ContainerResource.Container)
		case "memory":
			expressionQuery = GetContainerMemUsageExpression(namespace, name, kind, metric.ContainerResource.Container)
		}
	case autoscalingv2.PodsMetricSourceType:
		var labels []string
		if metric.Pods.Metric.Selector != nil {
			for k, v := range metric.Pods.Metric.Selector.MatchLabels {
				labels = append(labels, k+"="+`"`+v+`"`)
			}
		}
		expressionQuery = GetCustomerExpression(metric.Pods.Metric.Name, strings.Join(labels, ","))
	case autoscalingv2.ExternalMetricSourceType:
		var labels []string
		if metric.External.Metric.Selector != nil {
			for k, v := range metric.External.Metric.Selector.MatchLabels {
				labels = append(labels, k+"="+`"`+v+`"`)
			}
		}
		expressionQuery = GetCustomerExpression(metric.External.Metric.Name, strings.Join(labels, ","))
	}

	return expressionQuery
}
