package utils

import (
	"fmt"
	"regexp"
	"strings"
	"encoding/json"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"

	"github.com/gocrane/crane/pkg/known"
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
	case autoscalingv2.ResourceMetricSourceType, autoscalingv2.PodsMetricSourceType, autoscalingv2.ExternalMetricSourceType:
		metricName = known.MetricNamePrediction
	}

	return metricName
}

// GetCronMetricName return metric name used by cron
func GetCronMetricName() string {
	return known.MetricNameCron
}

// GetGeneralPredictionMetricName return metric name used by prediction
func GetMetricIdentifier(metric autoscalingv2.MetricSpec, name string) string {
	var prefix string
	switch metric.Type {
	case autoscalingv2.PodsMetricSourceType:
		prefix = "pods"
	case autoscalingv2.ResourceMetricSourceType:
		prefix = "resource"
	case autoscalingv2.ExternalMetricSourceType:
		prefix = "external"
	}

	return fmt.Sprintf("%s.%s", prefix, name)
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

// GetExpressionQuery return metric query
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

// GetAnnotationPromAdapter return value from annotation by suffix
func GetAnnotationPromAdapter(identifier string, annotations map[string]string) string {
	for k, v := range annotations {
		if strings.HasPrefix(k, known.EffectiveHorizontalPodAutoscalerAnnotationPromAdapter) {
			compileRegex := regexp.MustCompile(fmt.Sprintf("%s(.*)", known.EffectiveHorizontalPodAutoscalerAnnotationPromAdapter))
			matchArr := compileRegex.FindStringSubmatch(k)
			if len(matchArr) == 2 && matchArr[1][1:] == identifier {
				return v
			}
		}
	}

	return ""
}

// GetExtensionLabelsAnnotationPromAdapter return match labels for prometheus adapter
func GetExtensionLabelsAnnotationPromAdapter(annotations map[string]string) (map[string]string, error) {
	var extensionLabels = make(map[string]string)
	var err error

	value := GetAnnotationPromAdapter(known.PromAdapterExtensionLabels, annotations)
	if value == "" {
		return extensionLabels, err
	}

	var labels = []map[string]interface{}{}
	err = json.Unmarshal([]byte(value), &labels)
	if err != nil {
		return extensionLabels, err
	}

	for _, label := range labels {
		for k := range label {
			extensionLabels[k] = fmt.Sprintf("%v", label[k])
		}
	}
	return extensionLabels, err
}
