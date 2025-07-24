package accuracy

import (
	"fmt"
	"math"
)

// PredictionError returns prediction error.
// In case MAPE returns error, MAE is used as fallback.
func PredictionError(actual, predicted []float64) (float64, error) {
	pe, err := MAPE(actual, predicted)
	if err != nil {
		return MAE(actual, predicted)
	}
	return pe, nil
}

// MAPE - Mean Absolute Percentage Error
func MAPE(actual, predicted []float64) (float64, error) {
	if len(actual) != len(predicted) {
		return 0., fmt.Errorf("actual and predicted series are not of the same length")
	}

	var e float64

	n := len(actual)
	var epsilon float64 = 1e-3
	for i := 0; i < n; i++ {
		if actual[i] < epsilon {
			return 0, fmt.Errorf("actual value(%f) is too close to zero", actual[i])
		}
		if predicted[i] < actual[i] {
			// If the predicted value is less than the actual one, we amplify the error
			e += amplify((actual[i] - predicted[i]) / actual[i])
		} else {
			e += (predicted[i] - actual[i]) / actual[i]
		}
	}
	e = e / float64(n)

	return e, nil
}

// Amplify x (0.0 < x < 1.0). The bigger x the greater the degree of amplification.
// For example, amplify(0.1) = 0.47 (+370%), amplify(0.5) = 3.1 (+520%)
func amplify(x float64) float64 {
	return -math.Log(1.0-x) / math.Log(1.25)
}

// MAE - Mean Absolute Error
func MAE(actual, predicted []float64) (float64, error) {
	if len(actual) != len(predicted) {
		return 0., fmt.Errorf("actual and predicted series are not the same length")
	}

	var e float64

	n := len(actual)

	for i := 0; i < n; i++ {
		e += math.Abs(actual[i] - predicted[i])
	}
	e = e / float64(n)

	return e, nil
}
