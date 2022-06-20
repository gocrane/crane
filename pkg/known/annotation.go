package known

const (
	HPARecommendationValueAnnotation      = "analysis.crane.io/hpa-recommendation"
	ReplicasRecommendationValueAnnotation = "analysis.crane.io/replicas-recommendation"
	ResourceRecommendationValueAnnotation = "analysis.crane.io/resource-recommendation"
)

const (
	EffectiveHorizontalPodAutoscalerCurrentMetricsAnnotation = "autoscaling.crane.io/effective-hpa-current-metrics"
	EffectiveHorizontalPodAutoscalerExternalMetricsAnnotationPrefix = "metric-name.autoscaling.crane.io"
)
