package prediction

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/scale"
	"k8s.io/klog/v2"

	craneclientset "github.com/gocrane/api/pkg/generated/clientset/versioned"
	"github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/controller/timeseriesprediction"
	"github.com/gocrane/crane/pkg/prediction/dsp"
	"github.com/gocrane/crane/pkg/prediction/percentile"
	predictormgr "github.com/gocrane/crane/pkg/predictor"
	"github.com/gocrane/crane/pkg/server/config"
	"github.com/gocrane/crane/pkg/server/ginwrapper"
	"github.com/gocrane/crane/pkg/utils/target"
)

const (
	day       = time.Hour * 24
	threeDays = time.Hour * 24 * 3
	week      = time.Hour * 24 * 7
)

type DebugHandler struct {
	craneClient      *craneclientset.Clientset
	predictorManager predictormgr.Manager
	selectorFetcher  target.SelectorFetcher
}

type ContextKey string

var (
	PredictorManagerKey ContextKey = "predictorManager"
	SelectorFetcherKey  ContextKey = "selectorFetcher"
)

func NewDebugHandler(config *config.Config) *DebugHandler {
	discoveryClientSet, err := discovery.NewDiscoveryClientForConfig(config.KubeConfig)
	if err != nil {
		klog.Exit(err, "Unable to create discover client")
	}

	scaleKindResolver := scale.NewDiscoveryScaleKindResolver(discoveryClientSet)
	scaleClient := scale.New(discoveryClientSet.RESTClient(), config.RestMapper, dynamic.LegacyAPIPathResolverFunc, scaleKindResolver)
	selectorFetcher := target.NewSelectorFetcher(config.Scheme, config.RestMapper, scaleClient, config.Client)

	return &DebugHandler{
		craneClient:      craneclientset.NewForConfigOrDie(config.KubeConfig),
		predictorManager: config.PredictorMgr,
		selectorFetcher:  selectorFetcher,
	}
}

func (dh *DebugHandler) Display(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("tsp")
	if len(namespace) == 0 || len(name) == 0 {
		c.Writer.WriteHeader(http.StatusBadRequest)
		return
	}

	tsp, err := dh.craneClient.PredictionV1alpha1().TimeSeriesPredictions(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}

	if len(tsp.Spec.PredictionMetrics) > 0 {
		if tsp.Spec.PredictionMetrics[0].Algorithm.AlgorithmType == v1alpha1.AlgorithmTypeDSP && tsp.Spec.PredictionMetrics[0].Algorithm.DSP != nil {
			mc, err := timeseriesprediction.NewMetricContext(dh.selectorFetcher, tsp, dh.predictorManager)
			if err != nil {
				ginwrapper.WriteResponse(c, err, nil)
				return
			}

			internalConf := mc.ConvertApiMetric2InternalConfig(&tsp.Spec.PredictionMetrics[0])
			namer := mc.GetMetricNamer(&tsp.Spec.PredictionMetrics[0])
			pred := dh.predictorManager.GetPredictor(v1alpha1.AlgorithmTypeDSP)
			history, test, estimate, err := dsp.Debug(pred, namer, internalConf)
			if err != nil {
				ginwrapper.WriteResponse(c, err, nil)
				return
			}

			dspMAEDay := 0.0
			daySampleLength := int(estimate.SampleRate * day.Seconds())
			if len(estimate.Samples) >= daySampleLength {
				dspMAEDay = mae(test.Samples[:daySampleLength-1], estimate.Samples[:daySampleLength-1])
			}
			dspMAEThreeDays := 0.0
			threeDaySampleLength := int(estimate.SampleRate * threeDays.Seconds())
			if len(estimate.Samples) >= threeDaySampleLength {
				dspMAEThreeDays = mae(test.Samples[:threeDaySampleLength-1], estimate.Samples[:threeDaySampleLength-1])
			}
			dspMAEWeekly := 0.0
			weeklyDaySampleLength := int(estimate.SampleRate * week.Seconds())
			if len(estimate.Samples) >= weeklyDaySampleLength {
				dspMAEWeekly = mae(test.Samples[:weeklyDaySampleLength-1], estimate.Samples[:weeklyDaySampleLength-1])
			}
			klog.Infof("dspMAE1d: %f,dspMAE3d: %f, dspMAE7d: %f", dspMAEDay, dspMAEThreeDays, dspMAEWeekly)
			page := components.NewPage()
			page.AddCharts(plot(history, "history", "green", charts.WithTitleOpts(opts.Title{Title: "history"})))
			page.AddCharts(plots([]*dsp.Signal{test, estimate}, []string{"actual", "forecasted"},
				charts.WithTitleOpts(opts.Title{Title: "actual/forecasted"})))

			page.AddCharts(bar(dspMAEDay, dspMAEThreeDays, dspMAEWeekly))
			err = page.Render(c.Writer)
			if err != nil {
				klog.ErrorS(err, "Failed to display debug time series")
			}

			return
		} else if tsp.Spec.PredictionMetrics[0].Algorithm.AlgorithmType == v1alpha1.AlgorithmTypePercentile && tsp.Spec.PredictionMetrics[0].Algorithm.Percentile != nil {
			mc, err := timeseriesprediction.NewMetricContext(dh.selectorFetcher, tsp, dh.predictorManager)
			if err != nil {
				ginwrapper.WriteResponse(c, err, nil)
				return
			}

			internalConf := mc.ConvertApiMetric2InternalConfig(&tsp.Spec.PredictionMetrics[0])
			internalConf.Percentile.HistoryLength = "7d"
			namer := mc.GetMetricNamer(&tsp.Spec.PredictionMetrics[0])
			predictor := dh.predictorManager.GetPredictor(v1alpha1.AlgorithmTypePercentile)
			history, estimate, err := percentile.Debug(predictor, namer, internalConf)
			if err != nil {
				ginwrapper.WriteResponse(c, err, nil)
				return
			}
			if len(history) <= 0 || len(estimate) <= 0 {
				c.Writer.WriteHeader(http.StatusBadRequest)
				return
			}

			samples := history[0].Samples
			estimateSamples := estimate[0].Samples
			historySignal := &dsp.Signal{SampleRate: 1 / 60.0}
			estimateSignal := &dsp.Signal{SampleRate: 1 / 60.0}
			for _, v := range samples {
				historySignal.Samples = append(historySignal.Samples, v.Value)
				estimateSignal.Samples = append(estimateSignal.Samples, estimateSamples[0].Value)
			}

			predictMAEDay := 0.0
			daySampleLength := int(estimateSignal.SampleRate * day.Seconds())
			if len(estimateSignal.Samples) >= daySampleLength {
				m := maxValue(historySignal.Samples[:daySampleLength-1])
				predictMAEDay = mae([]float64{m}, []float64{estimateSamples[0].Value})
			}
			predictMAEThreeDays := 0.0
			threeDaySampleLength := int(estimateSignal.SampleRate * threeDays.Seconds())
			if len(estimateSignal.Samples) >= threeDaySampleLength {
				m := maxValue(historySignal.Samples[:threeDaySampleLength-1])
				predictMAEThreeDays = mae([]float64{m}, []float64{estimateSamples[0].Value})
			}
			predictMAEWeekly := 0.0
			weeklySampleLength := int(estimateSignal.SampleRate * week.Seconds())
			if len(estimateSignal.Samples) >= weeklySampleLength {
				m := maxValue(historySignal.Samples[:weeklySampleLength-1])
				predictMAEWeekly = mae([]float64{m}, []float64{estimateSamples[0].Value})
			}

			page := components.NewPage()
			page.AddCharts(plots([]*dsp.Signal{historySignal, estimateSignal}, []string{"actual", "recommended"},
				charts.WithTitleOpts(opts.Title{Title: "actual/recommended"})))

			page.AddCharts(bar(predictMAEDay, predictMAEThreeDays, predictMAEWeekly))
			err = page.Render(c.Writer)
			if err != nil {
				klog.ErrorS(err, "Failed to display debug time series")
			}

			c.Writer.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	c.Writer.WriteHeader(http.StatusBadRequest)
	return
}

func maxValue(testData []float64) float64 {
	m := 0.0
	for i := 0; i < len(testData); i++ {
		if testData[i] > m {
			m = testData[i]
		}
	}
	return m
}

func mae(testData, estimateData []float64) float64 {
	mae := 0.0
	for i := 0; i < len(estimateData); i++ {
		mae += math.Abs(testData[i] - estimateData[i])
	}
	return mae / float64(len(estimateData))
}

func bar(dspMAE1d, dspMAE3d, dspMAE7d float64) *charts.Bar {
	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Width: "3000px", Theme: types.ThemeRoma}),
		charts.WithTitleOpts(opts.Title{
			Title: "MAE(Mean Absolute Error)",
		}), charts.WithTooltipOpts(opts.Tooltip{
			Show:      true,
			Trigger:   "axis",
			TriggerOn: "mousemove",
		}),
	)
	bar.SetXAxis([]string{"1d", "3d", "7d"}).
		AddSeries("mae", []opts.BarData{
			{
				Value: dspMAE1d,
			},
			{
				Value: dspMAE3d,
			},
			{
				Value: dspMAE7d,
			},
		})
	return bar
}

func plot(s *dsp.Signal, name string, color string, o ...charts.GlobalOpts) *charts.Line {
	x := make([]string, 0)
	y := make([]opts.LineData, 0)
	for i := 0; i < s.Num(); i++ {
		x = append(x, fmt.Sprintf("%.1f", float64(i)/s.SampleRate))
		y = append(y, opts.LineData{Value: s.Samples[i], Symbol: "none"})
	}

	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Width: "3000px", Theme: types.ThemeRoma}),
		charts.WithLegendOpts(
			opts.Legend{
				Show: true,
				Data: name,
			}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:      true,
			Trigger:   "axis",
			TriggerOn: "mousemove",
		}),
		charts.WithTitleOpts(opts.Title{Title: s.String()}))
	if o != nil {
		line.SetGlobalOptions(o...)
	}
	line.SetXAxis(x).AddSeries(name, y, charts.WithLineStyleOpts(opts.LineStyle{Color: color}))

	return line
}

func plots(signals []*dsp.Signal, names []string, o ...charts.GlobalOpts) *charts.Line {
	if len(signals) < 1 {
		return nil
	}
	s := signals[0]
	n := signals[0].Num()
	x := make([]string, 0)
	y := make([][]opts.LineData, len(signals))
	for j := 0; j < len(signals); j++ {
		y[j] = make([]opts.LineData, 0)
	}
	for i := 0; i < n; i++ {
		x = append(x, fmt.Sprintf("%.1f", float64(i)/s.SampleRate))
		for j := 0; j < len(signals); j++ {
			y[j] = append(y[j], opts.LineData{Value: signals[j].Samples[i], Symbol: "none"})
		}
	}

	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Width: "3000px", Theme: types.ThemeRoma}),
		charts.WithLegendOpts(
			opts.Legend{
				Show: true,
				Data: names,
			}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:      true,
			Trigger:   "axis",
			TriggerOn: "mousemove",
		}))
	if o != nil {
		line.SetGlobalOptions(o...)
	}
	line.SetXAxis(x)
	for j := 0; j < len(signals); j++ {
		line.AddSeries(names[j], y[j], charts.WithAreaStyleOpts(
			opts.AreaStyle{
				Opacity: 0.1,
			}),
		)
	}
	return line
}
