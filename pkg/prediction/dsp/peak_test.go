package dsp

import (
	"testing"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
)

func TestLinerRegression(t *testing.T) {
	var s, _ = readCsvFile("test_data/input8.csv")
	s.Samples = s.Samples[:1440*14]
	cor := AutoCorrelation(s.Samples)

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

	xAxis = make([]int, 0)
	yAxis = make([]opts.LineData, 0)
	for i := 1440*7 - 240; i < 1440*7+240; i++ {
		xAxis = append(xAxis, i)
		yAxis = append(yAxis, opts.LineData{Value: cor[i], Symbol: "cycle"})
	}
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Width: "1000px", Theme: types.ThemeMacarons}),
		charts.WithTitleOpts(opts.Title{Title: ""}))
	line.SetXAxis(xAxis).AddSeries("", yAxis)

	points := []point{}
	xAxis = make([]int, 0)
	yAxis = make([]opts.LineData, 0)

	// left
	for i := 1440*7 - 240; i < 1440*7; i++ {
		points = append(points, point{x: float64(i), y: cor[i]})
		xAxis = append(xAxis, i)
	}
	a, b := linearRegressionLSE(points)
	for i := range xAxis {
		yAxis = append(yAxis, opts.LineData{Value: a*float64(xAxis[i]) + b, Symbol: "cycle"})
	}

	// right
	points = []point{}
	for i := 1440 * 7; i < 1440*7+240; i++ {
		points = append(points, point{x: float64(i), y: cor[i]})
		xAxis = append(xAxis, i)
	}
	a, b = linearRegressionLSE(points)
	for i := range xAxis[240:] {
		yAxis = append(yAxis, opts.LineData{Value: a*float64(xAxis[240+i]) + b, Symbol: "cycle"})
	}

	linerLine := charts.NewLine()
	linerLine.SetXAxis(xAxis).AddSeries("", yAxis)
	line.Overlap(linerLine)

	/*
		Uncomment code below to see what auto-correlation function and its liner regression look like
	*/
	//http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
	//	page := components.NewPage()
	//	page.AddCharts(s.Plot("green"), acfLine, line)
	//	page.Render(w)
	//})
	//fmt.Println("Open your browser and access 'http://localhost:7001'")
	//http.ListenAndServe(":7001", nil)
}

func TestLinerRegression2(t *testing.T) {
	var s, _ = readCsvFile("test_data/input10.csv")
	cor := AutoCorrelation(s.Samples)

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

	xAxis = make([]int, 0)
	yAxis = make([]opts.LineData, 0)
	for i := 1440 - 240; i < 1440+240; i++ {
		xAxis = append(xAxis, i)
		yAxis = append(yAxis, opts.LineData{Value: cor[i], Symbol: "cycle"})
	}
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Width: "1000px", Theme: types.ThemeMacarons}),
		charts.WithTitleOpts(opts.Title{Title: ""}))
	line.SetXAxis(xAxis).AddSeries("", yAxis)

	points := []point{}
	xAxis = make([]int, 0)
	yAxis = make([]opts.LineData, 0)

	// left
	for i := 1440 - 240; i < 1440; i++ {
		points = append(points, point{x: float64(i), y: cor[i]})
		xAxis = append(xAxis, i)
	}
	a, b := linearRegressionLSE(points)
	for i := range xAxis {
		yAxis = append(yAxis, opts.LineData{Value: a*float64(xAxis[i]) + b, Symbol: "cycle"})
	}

	// right
	points = []point{}
	for i := 1440; i < 1440+240; i++ {
		points = append(points, point{x: float64(i), y: cor[i]})
		xAxis = append(xAxis, i)
	}
	a, b = linearRegressionLSE(points)
	for i := range xAxis[240:] {
		yAxis = append(yAxis, opts.LineData{Value: a*float64(xAxis[240+i]) + b, Symbol: "cycle"})
	}

	linerLine := charts.NewLine()
	linerLine.SetXAxis(xAxis).AddSeries("", yAxis)
	line.Overlap(linerLine)

	/*
		Uncomment code below to see what auto-correlation function and its liner regression look like
	*/
	//http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
	//	page := components.NewPage()
	//	page.AddCharts(s.Plot("green"), acfLine, line)
	//	page.Render(w)
	//})
	//fmt.Println("Open your browser and access 'http://localhost:7001'")
	//http.ListenAndServe(":7001", nil)
}
