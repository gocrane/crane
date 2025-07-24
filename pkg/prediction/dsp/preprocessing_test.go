package dsp

import (
	"math/rand"
	"testing"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/gocrane/crane/pkg/common"
	"github.com/stretchr/testify/assert"
)

func TestRemoveExtremeOutliers(t *testing.T) {
	n := 10080

	ts := &common.TimeSeries{
		Samples: make([]common.Sample, n),
	}

	rand.Seed(time.Now().UnixNano())
	for i := 0; i < n; i++ {
		ts.Samples[i] = common.Sample{
			Value: rand.Float64(),
		}
	}

	ts.Samples[1000].Value = 3.5
	ts.Samples[3000].Value = -1
	ts.Samples[8000].Value = 1000

	_ = removeExtremeOutliers(ts)

	assert.Equal(t, ts.Samples[999].Value, ts.Samples[1000].Value)
	assert.Equal(t, ts.Samples[2999].Value, ts.Samples[3000].Value)
	assert.Equal(t, ts.Samples[7999].Value, ts.Samples[8000].Value)
}

func TestRemoveExtremeOutliers2(t *testing.T) {
	var s, _ = readCsvFile("test_data/input14.csv")
	origLine := s.Plot("green")

	ts := &common.TimeSeries{
		Samples: make([]common.Sample, len(s.Samples)),
	}
	for i := 0; i < len(s.Samples); i++ {
		ts.Samples[i].Value = s.Samples[i]
	}
	_ = removeExtremeOutliers(ts)
	xAxis := make([]int, 0)
	yAxis := make([]opts.LineData, 0)
	for i := range ts.Samples {
		xAxis = append(xAxis, i)
		yAxis = append(yAxis, opts.LineData{Value: ts.Samples[i].Value, Symbol: "cycle"})
	}
	line := charts.NewLine()
	line.SetXAxis(xAxis).AddSeries("", yAxis)

	origLine.Overlap(line)
	/*
			Uncomment code below to see what the original signal and the one after
		    getting removed outliers look like
	*/
	//http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
	//	page := components.NewPage()
	//	page.AddCharts(origLine)
	//	page.Render(w)
	//})
	//fmt.Println("Open your browser and access 'http://localhost:7001'")
	//http.ListenAndServe(":7001", nil)
}
