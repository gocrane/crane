package dsp

import (
	"math"
	"testing"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/stretchr/testify/assert"
)

func TestAutoCorrelation(t *testing.T) {
	epsilon := 1e-15
	sampleRate := 1000.0               // Hz
	sampleInterval := 1.0 / sampleRate // seconds
	var x []float64
	for i := 0.0; i < 1; i += sampleInterval {
		x = append(x, i)
	}
	frequency := 5.0

	y := make([]float64, len(x))
	for i := range x {
		y[i] = math.Sin(2.*math.Pi*frequency*x[i]) +
			(1./3.)*math.Sin(3.*2.*math.Pi*frequency*x[i]) +
			(1./5.)*math.Sin(5.*2.*math.Pi*frequency*x[i]) +
			(1./7.)*math.Sin(7.*2.*math.Pi*frequency*x[i])

	}
	sine := &Signal{
		SampleRate: sampleRate,
		Samples:    y,
	}

	cor := AutoCorrelation(sine.Samples)
	assert.InEpsilon(t, -1, cor[100], epsilon)
	assert.InEpsilon(t, 1, cor[200], epsilon)

	xAxis := make([]int, 0)
	yAxis := make([]opts.LineData, 0)
	for i := range cor {
		xAxis = append(xAxis, i)
		yAxis = append(yAxis, opts.LineData{Value: cor[i], Symbol: "cycle"})
	}

	acfLine := charts.NewLine()
	acfLine.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Width: "3000px", Theme: types.ThemeChalk}),
		charts.WithTitleOpts(opts.Title{Title: "Auto Correlation"}))
	acfLine.SetXAxis(xAxis).AddSeries("Auto Correlation", yAxis)

	/*
		Uncomment code below to see what the signal and its auto-correlation function look like
	*/
	//http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
	//	page := components.NewPage()
	//	page.AddCharts(sine.Plot(), acfLine)
	//	page.Render(w)
	//})
	//fmt.Println("Open your browser and access 'http://localhost:7001'")
	//http.ListenAndServe(":7001", nil)
}
