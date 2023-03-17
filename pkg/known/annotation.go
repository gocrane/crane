package known

const (
	HPARecommendationValueAnnotation      = "analysis.crane.io/hpa-recommendation"
	ReplicasRecommendationValueAnnotation = "analysis.crane.io/replicas-recommendation"
	ResourceRecommendationValueAnnotation = "analysis.crane.io/resource-recommendation"
	RunNumberAnnotation                   = "analysis.crane.io/run-number"
)

const (
	EffectiveHorizontalPodAutoscalerCurrentMetricsAnnotation        = "autoscaling.crane.io/effective-hpa-current-metrics"
	EffectiveHorizontalPodAutoscalerExternalMetricsAnnotationPrefix = "metric-query.autoscaling.crane.io"
)
