package logic

type OpaLogic struct {

}

func NewOpaLogic() Logic{
	return &OpaLogic{}
}

func (c *OpaLogic) EvalWithMetric(metricName string, targetValue float64, value float64) (bool, error) {

	// step1 splice policy and input strings
	// step2 opa.New().WithPolicyBytes
	// step3 rego.Eval
	// step4 transfer opa.result to bool
	// return

	return false, nil
}

func (c *OpaLogic) EvalWithRaw(input string, rule string) (bool, error) {
	return false, nil
}
