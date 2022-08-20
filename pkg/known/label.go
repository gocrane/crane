package known

const (
	EffectiveHorizontalPodAutoscalerUidLabel  = "autoscaling.crane.io/effective-hpa-uid"
	EffectiveHorizontalPodAutoscalerManagedBy = "effective-hpa-controller"
)

const (
	EnsuranceAnalyzedPressureTaintKey     = "interference.crane.io"
	EnsuranceAnalyzedPressureConditionKey = "interference-identified"
)

const (
	AnalyticsNameLabel = "analysis.crane.io/analytics-name"
	AnalyticsUidLabel  = "analysis.crane.io/analytics-uid"
	AnalyticsTypeLabel = "analysis.crane.io/analytics-type"
)

const (
	RecommendationRuleNameLabel        = "analysis.crane.io/recommendation-rule-name"
	RecommendationRuleUidLabel         = "analysis.crane.io/recommendation-rule-uid"
	RecommendationRuleRecommenderLabel = "analysis.crane.io/recommendation-rule-recommender"
)
