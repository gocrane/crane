package logic

type Logic interface {
	EvalWithMetric(metricName string, targetValue float64, value float64) (bool, error)
	EvalWithRaw(input string, rule string) (bool, error)
}