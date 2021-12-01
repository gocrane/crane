package dsp

import (
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/stretchr/testify/assert"
	"math"
	"math/rand"
	"testing"
)

func TestSignal_Period(t *testing.T) {
	epsilon := 1e-15
	sampleRate := 1000.0               // Hz
	sampleInterval := 1.0 / sampleRate // seconds
	var x []float64
	for i := 0.0; i < 1; i += sampleInterval {
		x = append(x, i)
	}

	frequency := 5.0

	expected := []float64{
		frequency,
		frequency,
		frequency,
		frequency,
		frequency,
	}

	f := []func(float64) float64{
		func(x float64) float64 {
			return math.Sin(2.0*math.Pi*frequency*x) + 1.0
		},
		func(x float64) float64 {
			return math.Sin(2.0*math.Pi*frequency*x+rand.Float64()) + 1.0
		},
		func(x float64) float64 {
			return math.Sin(2.0*math.Pi*frequency*x) + 0.5*math.Sin(2.0*math.Pi*4*frequency*x) + 3
		},
		func(x float64) float64 {
			return math.Sin(2.0*math.Pi*frequency*x) + rand.Float64()*2.0
		},
		func(x float64) float64 {
			growthRatio := 0.1
			period := 1.0 / frequency
			return (1.0 + math.Floor(x/period)*growthRatio) * math.Sin(2.0*math.Pi*frequency*x)
		},
	}

	var lines []*charts.Line

	for i := range expected {
		var y []float64
		for j := range x {
			y = append(y, f[i](x[j]))
		}
		s := &Signal{
			SampleRate: sampleRate,
			Samples:    y,
		}
		normalized, err := s.Normalize()
		assert.NoError(t, err)
		assert.InEpsilon(t, expected[i], normalized.Frequencies()[0], epsilon)

		lines = append(lines, s.Plot())
	}

	/*
		Uncomment code below to see what above signals look like
	*/
	//http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
	//	page := components.NewPage()
	//	for i := range lines {
	//		page.AddCharts(lines[i])
	//	}
	//	page.Render(w)
	//})
	//fmt.Println("Open your browser and access 'http://localhost:7001'")
	//http.ListenAndServe(":7001", nil)
}
