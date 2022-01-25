package evaluator

type ExpressionEvaluator struct {
}

func NewExpressionEvaluator() Evaluator {
	return &ExpressionEvaluator{}
}

func (c *ExpressionEvaluator) EvalWithMetric(metricName string, targetValue float64, value float64) bool {
	return value > targetValue
}

func (c *ExpressionEvaluator) EvalWithRawQuery(input string, rule string) bool {
	return false
}
