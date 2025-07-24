package accuracy

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	epsilon        = 1e-10
	sampleRate     = 100.0            // Hz
	sampleInterval = 1.0 / sampleRate // seconds
)

func TestMAPE(t *testing.T) {
	var x, y, yp []float64
	for i := 0.0; i < 1; i += sampleInterval {
		x = append(x, i)
	}
	for i := range x {
		y = append(y, math.Sin(2.0*math.Pi*2.0*x[i])+3.0)
	}
	diffFraction := .10
	for i := range x {
		yp = append(yp, y[i]*(1.0+diffFraction))
	}

	mape, err := MAPE(y, y)
	assert.NoError(t, err)
	assert.Equal(t, 0.0, mape)

	mape, err = MAPE(y, yp)
	fmt.Println(mape)
	assert.NoError(t, err)
	assert.InEpsilon(t, diffFraction, mape, epsilon)

	yp = yp[:0]
	for i := range x {
		yp = append(yp, y[i]*(1.0-diffFraction))
	}

	mape, err = MAPE(y, yp)
	fmt.Println(mape)
	assert.NoError(t, err)
	assert.Less(t, diffFraction, mape)
	assert.InEpsilon(t, amplify(diffFraction), mape, epsilon)
}

func TestMAE(t *testing.T) {
	var x, y, yp []float64
	for i := 0.0; i < 1; i += sampleInterval {
		x = append(x, i)
	}
	for i := range x {
		y = append(y, math.Sin(2.0*math.Pi*2.0*x[i])+3.0)
	}
	diffFraction := .10
	for i := range x {
		yp = append(yp, y[i]*(1.0+diffFraction))
	}
	mae, err := MAE(y, y)
	assert.NoError(t, err)
	assert.Equal(t, 0.0, mae)

	mae, err = MAE(y, yp)
	assert.NoError(t, err)
	fmt.Println(mae)
}
