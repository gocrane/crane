package percentile

import (
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/recommender/util"
)

type Estimator interface {
	GetEstimation(h vpa.Histogram) float64
}

type percentileEstimator struct {
	percentile float64
}

type marginEstimator struct {
	marginFraction float64
	baseEstimator  Estimator
}

type targetUtilizationEstimator struct {
	targetUtilization float64
	baseEstimator     Estimator
}

func NewPercentileEstimator(percentile float64) Estimator {
	return &percentileEstimator{percentile}
}

func WithMargin(marginFraction float64, baseEstimator Estimator) Estimator {
	return &marginEstimator{marginFraction, baseEstimator}
}

func WithTargetUtilization(targetUtilization float64, baseEstimator Estimator) Estimator {
	return &targetUtilizationEstimator{targetUtilization, baseEstimator}
}

func (e *percentileEstimator) GetEstimation(h vpa.Histogram) float64 {
	return h.Percentile(e.percentile)
}

func (e *marginEstimator) GetEstimation(h vpa.Histogram) float64 {
	return e.baseEstimator.GetEstimation(h) * (1 + e.marginFraction)
}

func (e *targetUtilizationEstimator) GetEstimation(h vpa.Histogram) float64 {
	return e.baseEstimator.GetEstimation(h) / e.targetUtilization
}
