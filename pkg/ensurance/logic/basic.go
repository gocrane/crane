package logic

type BasicLogic struct {
}

func NewBasicLogic() Logic {
	return &BasicLogic{}
}

func (c *BasicLogic) EvalWithMetric(metricName string, targetValue float64, value float64) (bool, error) {
	return value > targetValue, nil
}

func (c *BasicLogic) EvalWithRaw(input string, rule string) (bool, error) {
	return false, nil
}
