package prediction

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/gocrane/crane/pkg/prediction/dsp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"net/http"

	craneclientset "github.com/gocrane/api/pkg/generated/clientset/versioned"
	"github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/controller/timeseriesprediction"
	predictormgr "github.com/gocrane/crane/pkg/predictor"
	"github.com/gocrane/crane/pkg/server/ginwrapper"
	"github.com/gocrane/crane/pkg/utils/target"
)

type DebugHandler struct {
	craneClient *craneclientset.Clientset
	predictorManager predictormgr.Manager
	selectorFetcher target.SelectorFetcher
}

func NewDebugHandler(ctx context.Context) *DebugHandler {
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatalf("Failed to get InClusterConfig, %v.", err)
	}

	val := ctx.Value("predictorManager")
	if val == nil {
		klog.Fatalf("predictorManager not found")
	}
	predictorManager := val.(predictormgr.Manager)

	val = ctx.Value("selectorFetcher")
	if val == nil {
		klog.Fatalf("selectorFetcher not found")
	}
	selectorFetcher := val.(target.SelectorFetcher)

	return &DebugHandler{
		craneClient: craneclientset.NewForConfigOrDie(config),
		predictorManager: predictorManager,
		selectorFetcher: selectorFetcher,
	}
}

func (dh *DebugHandler) Display(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("tsp")
	klog.Infof("WWWW Display %s/%s.", namespace, name)

	if len(namespace) == 0 || len(name) == 0 {
		c.Writer.WriteHeader(http.StatusBadRequest)
		return
	}

	tsp, err := dh.craneClient.PredictionV1alpha1().TimeSeriesPredictions(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}
	klog.Infof("WWWW Got TimeSeriesPrediction %s/%s.", tsp.Namespace, tsp.Name)

	if len(tsp.Spec.PredictionMetrics) > 0 {
		if tsp.Spec.PredictionMetrics[0].Algorithm.AlgorithmType == v1alpha1.AlgorithmTypeDSP && tsp.Spec.PredictionMetrics[0].Algorithm.DSP != nil  {
			mc, err := timeseriesprediction.NewMetricContext(dh.selectorFetcher, tsp, dh.predictorManager)
			if err != nil {
				ginwrapper.WriteResponse(c, err, nil)
				return
			}
klog.Infof("WWWW MetricContext: %v", mc)
			internalConf := mc.ConvertApiMetric2InternalConfig(&tsp.Spec.PredictionMetrics[0])
klog.Infof("WWWW InternalConf: %v", *internalConf)
			namer := mc.GetMetricNamer(&tsp.Spec.PredictionMetrics[0])
klog.Infof("WWWW namer: %v", namer)
			pred := dh.predictorManager.GetPredictor(v1alpha1.AlgorithmTypeDSP)
			//gp := (*prediction.GenericPrediction)(unsafe.Pointer(p))
klog.Infof("WWWWWW p: %v", pred)
			history, test, estimate, err := dsp.Debug(pred, namer, internalConf)
			if err != nil {
				ginwrapper.WriteResponse(c, err, nil)
				return
			}

	klog.Infof("WWWWWWWWWWWWWWW\n")
			page := components.NewPage()
			page.AddCharts(history.Plot())
			page.AddCharts(plot([]*dsp.Signal{test, estimate}))
			page.Render(c.Writer)
			return
		}
	}

	c.Writer.WriteHeader(http.StatusBadRequest)
	return
}

func plot(signals []*dsp.Signal, o ...charts.GlobalOpts) *charts.Line {
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

	var colors []string = []string{"black", "blue", "green"}

	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Width: "3000px", Theme: types.ThemeRoma}),
		charts.WithTitleOpts(opts.Title{Title: s.String()}))
	if o != nil {
		line.SetGlobalOptions(o...)
	}
	line.SetXAxis(x)
	for j := 0; j < len(signals); j++ {
		line.AddSeries("s", y[j], charts.WithAreaStyleOpts(
			opts.AreaStyle{
				Color:   colors[j],
				Opacity: 0.1,
			}),
			charts.WithLineStyleOpts(opts.LineStyle{Color: colors[j]}))
	}
	return line
}