package evaluator

type Evaluator interface {
	EvalWithMetric(metricName string, targetValue float64, value float64) bool
	EvalWithRawQuery(input string, rule string) bool
}
