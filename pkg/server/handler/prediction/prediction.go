package prediction

import (
	"context"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/gocrane/crane/pkg/prediction"
	"github.com/gocrane/crane/pkg/prediction/dsp"
	"net/http"
	"unsafe"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

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
			p := dh.predictorManager.GetPredictor(v1alpha1.AlgorithmTypeDSP)
			gp := (*prediction.GenericPrediction)(unsafe.Pointer(&p))
klog.Infof("WWWWWW gp: %v", gp)
			history, test, estimate, err := dsp.Debug(gp, namer, internalConf)
			if err != nil {
				ginwrapper.WriteResponse(c, err, nil)
				return
			}

	klog.Infof("WWWWWWWWWWWWWWW\n")
			page := components.NewPage()
			page.AddCharts(history.Plot(), test.Plot(), estimate.Plot())
			page.Render(c.Writer)
			return
		}
	}

	c.Writer.WriteHeader(http.StatusBadRequest)
	return
}