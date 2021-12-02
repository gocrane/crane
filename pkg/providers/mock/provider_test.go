package mock

import (
	"context"
	"fmt"
	"github.com/gocrane/crane/pkg/prediction"
	"github.com/gocrane/crane/pkg/prediction/dsp"
	"github.com/gocrane/crane/pkg/providers"
	"net/http"
	"testing"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"

)

func TestMockProvider(t *testing.T) {
	prov, _ := NewProvider(&providers.MockConfig{SeedFile: "data.csv"})

	time.Sleep(time.Second)

	p, err := dsp.NewPrediction()
	if err != nil {
		panic(err)
	}

	p.WithProviders(map[string]providers.Interface{
		prediction.RealtimeProvider: prov,
		prediction.HistoryProvider:  prov,
	})

	//p.WithMetric("test", nil)
	err = p.WithQuery("test")
	if err != nil {
		panic(err)
	}

	ctx := context.TODO()
	go p.Run(ctx.Done())

	time.Sleep(3 * time.Second)

	now := time.Now()
	tsList, err := p.QueryPredictedTimeSeries("test", now, now.Add(24*time.Hour))
	if err != nil {
		panic(err)
	}

	ts := tsList[0]
	s := dsp.SamplesToSignal(ts.Samples, time.Minute)

	tsList, _ = prov.QueryTimeSeries("", now, now.Add(24*time.Hour), time.Minute)
	ts = tsList[0]
	orig := dsp.SamplesToSignal(ts.Samples, time.Minute)

	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		components.NewPage().AddCharts(plot(s, orig)).Render(w)
	})
	fmt.Println("Open your browser and access 'http://localhost:7001'")
	http.ListenAndServe(":7001", nil)
}

func plot(signals ...*dsp.Signal) *charts.Line {
	var colors []string = []string{"green", "blue", "red"}

	n := len(signals)
	if n < 1 {
		panic("no signal")
	}
	x := make([]string, 0)
	y := make([]opts.LineData, 0)
	s := signals[0]
	for i := 0; i < s.Num(); i++ {
		x = append(x, fmt.Sprintf("%.1f", float64(i)/s.SampleRate))
		//y = append(y, opts.LineData{Value: s.Samples[i], Symbol: "none"})
	}

	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Width: "3000px", Theme: types.ThemeWonderland}),
		charts.WithTitleOpts(opts.Title{Title: s.String()}))

	line.SetXAxis(x).AddSeries("predicted", y)

	for j := 0; j < n; j++ {
		y = make([]opts.LineData, 0)
		for i := 0; i < s.Num(); i++ {
			y = append(y, opts.LineData{Value: signals[j].Samples[i], Symbol: "none"})
		}
		line.AddSeries("s", y, charts.WithAreaStyleOpts(
			opts.AreaStyle{
				Color:   colors[j],
				Opacity: 0.2,
			}),
			charts.WithLineStyleOpts(opts.LineStyle{Color: colors[j]}))
	}

	return line
}
