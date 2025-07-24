package evaluator

type OpaEvaluator struct {
}

func NewOpaEvaluator() Evaluator {
	return &OpaEvaluator{}
}

func (c *OpaEvaluator) EvalWithMetric(metricName string, targetValue float64, value float64) bool {

	// step1 splice policy and input strings
	// step2 opa.New().WithPolicyBytes
	// step3 rego.Eval
	// step4 transfer opa.result to bool
	// return

	return false
}

func (c *OpaEvaluator) EvalWithRawQuery(input string, rule string) bool {
	return false
}
