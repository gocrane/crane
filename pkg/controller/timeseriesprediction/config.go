package timeseriesprediction

import (
	"fmt"
	"sort"
	"strings"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/metricquery"
	predconf "github.com/gocrane/crane/pkg/prediction/config"
	predictormgr "github.com/gocrane/crane/pkg/predictor"
	"github.com/gocrane/crane/pkg/utils/target"
)

type MetricContext struct {
	Namespace        string
	TargetKind       string
	Name             string
	APIVersion       string
	Selector         labels.Selector
	SeriesPrediction *predictionapi.TimeSeriesPrediction
	predictorMgr     predictormgr.Manager
	fetcher          target.SelectorFetcher
}

func NewMetricContext(fetcher target.SelectorFetcher, seriesPrediction *predictionapi.TimeSeriesPrediction, predictorMgr predictormgr.Manager) (*MetricContext, error) {
	c := &MetricContext{
		Namespace:        seriesPrediction.Namespace,
		TargetKind:       seriesPrediction.Spec.TargetRef.Kind,
		Name:             seriesPrediction.Spec.TargetRef.Name,
		APIVersion:       seriesPrediction.Spec.TargetRef.APIVersion,
		SeriesPrediction: seriesPrediction,
		predictorMgr:     predictorMgr,
		fetcher:          fetcher,
	}
	if strings.ToLower(c.TargetKind) != strings.ToLower(predconf.TargetKindNode) && seriesPrediction.Spec.TargetRef.Namespace != "" {
		c.Namespace = seriesPrediction.Spec.TargetRef.Namespace
	}
	if strings.ToLower(c.TargetKind) != strings.ToLower(predconf.TargetKindNode) {
		selector, err := c.fetcher.Fetch(&seriesPrediction.Spec.TargetRef)
		if err != nil {
			return nil, err
		}
		c.Selector = selector
	}

	return c, nil
}

func (c *MetricContext) GetCaller() string {
	return fmt.Sprintf(callerFormat, klog.KObj(c.SeriesPrediction), c.SeriesPrediction.UID)
}

func (c *MetricContext) GetMetricNamer(conf *predictionapi.PredictionMetric) metricnaming.MetricNamer {
	var namer metricnaming.GeneralMetricNamer
	if conf.MetricQuery != nil {
		klog.InfoS("GetQueryStr MetricQuery not supported", "tsp", klog.KObj(c.SeriesPrediction), "metricSelector", metricSelectorToQueryExpr(conf.MetricQuery))
		return nil
	}
	if conf.ExpressionQuery != nil {
		namer.Metric = &metricquery.Metric{
			Type:       metricquery.PromQLMetricType,
			MetricName: conf.ResourceIdentifier,
			Prom: &metricquery.PromNamerInfo{
				QueryExpr: conf.ExpressionQuery.Expression,
				Selector:  labels.Nothing(),
			},
		}
		klog.InfoS("GetQueryStr", "tsp", klog.KObj(c.SeriesPrediction), "queryExpr", conf.ExpressionQuery.Expression)
	}
	if conf.ResourceQuery != nil {
		namer = c.ResourceToMetricNamer(conf.ResourceQuery)
		klog.InfoS("GetQueryStr", "tsp", klog.KObj(c.SeriesPrediction), "resourceQuery", conf.ResourceQuery)
	}
	return &namer
}

func (c *MetricContext) WithApiConfig(conf *predictionapi.PredictionMetric) {
	if predictor := c.predictorMgr.GetPredictor(conf.Algorithm.AlgorithmType); predictor != nil {
		internalConf := c.ConvertApiMetric2InternalConfig(conf)
		namer := c.GetMetricNamer(conf)
		queryStr := namer.BuildUniqueKey()
		err := predictor.WithQuery(namer, c.GetCaller(), *internalConf)
		if err != nil {
			klog.InfoS("WithApiConfig WithQuery registered failed", "tsp", klog.KObj(c.SeriesPrediction), "queryStr", queryStr)
			return
		}
		klog.InfoS("WithApiConfig WithQuery registered succeed", "tsp", klog.KObj(c.SeriesPrediction), "queryStr", queryStr)
	} else {
		klog.InfoS("WithApiConfig predictor %v not found, ignore tsp %v", conf.Algorithm.AlgorithmType, klog.KObj(c.SeriesPrediction))
	}
}

func (c *MetricContext) WithApiConfigs(configs []predictionapi.PredictionMetric) {
	for _, conf := range configs {
		c.WithApiConfig(&conf)
	}
}

func (c *MetricContext) DeleteApiConfig(conf *predictionapi.PredictionMetric) {
	namer := c.GetMetricNamer(conf)
	queryStr := namer.BuildUniqueKey()
	klog.InfoS("DeleteApiConfig DeleteQuery", "tsp", klog.KObj(c.SeriesPrediction), "queryStr", queryStr)
	if predictor := c.predictorMgr.GetPredictor(conf.Algorithm.AlgorithmType); predictor != nil {
		err := predictor.DeleteQuery(namer, c.GetCaller())
		if err != nil {
			klog.InfoS("DeleteApiConfig DeleteQuery deleted failed", "tsp", klog.KObj(c.SeriesPrediction), "queryStr", queryStr)
			return
		}
		klog.InfoS("DeleteApiConfig DeleteQuery deleted succeed", "tsp", klog.KObj(c.SeriesPrediction), "queryStr", queryStr)
	} else {
		klog.InfoS("DeleteApiConfig predictor %v not found, ignore tsp %v", conf.Algorithm.AlgorithmType, klog.KObj(c.SeriesPrediction))
	}

}

func (c *MetricContext) DeleteApiConfigs(configs []predictionapi.PredictionMetric) {
	for _, conf := range configs {
		c.DeleteApiConfig(&conf)
	}
}

func metricSelectorToQueryExpr(m *predictionapi.MetricQuery) string {
	conditions := make([]string, 0, len(m.QueryConditions))
	for _, cond := range m.QueryConditions {
		values := make([]string, 0, len(cond.Value))
		for _, val := range cond.Value {
			values = append(values, val)
		}
		sort.Strings(values)
		conditions = append(conditions, fmt.Sprintf("%s%s[%s]", cond.Key, cond.Operator, strings.Join(values, ",")))
	}
	sort.Strings(conditions)
	return fmt.Sprintf("%s{%s}", m.MetricName, strings.Join(conditions, ","))
}

func (c *MetricContext) ResourceToMetricNamer(resourceName *corev1.ResourceName) metricnaming.GeneralMetricNamer {
	var namer metricnaming.GeneralMetricNamer

	// Node
	if strings.ToLower(c.TargetKind) == strings.ToLower(predconf.TargetKindNode) {
		namer.Metric = &metricquery.Metric{
			Type:       metricquery.NodeMetricType,
			MetricName: resourceName.String(),
			Node: &metricquery.NodeNamerInfo{
				Name:     c.Name,
				Selector: labels.Everything(),
			},
		}
	} else {
		// workload
		namer.Metric = &metricquery.Metric{
			Type:       metricquery.WorkloadMetricType,
			MetricName: resourceName.String(),
			Workload: &metricquery.WorkloadNamerInfo{
				Namespace:  c.Namespace,
				Kind:       c.TargetKind,
				APIVersion: c.APIVersion,
				Name:       c.Name,
				Selector:   c.Selector,
			},
		}
	}
	return namer
}

// ConvertApiMetrics2InternalConfigs
func (c *MetricContext) ConvertApiMetrics2InternalConfigs(metrics []predictionapi.PredictionMetric) []*predconf.Config {
	var confs []*predconf.Config
	for _, metric := range metrics {
		confs = append(confs, c.ConvertApiMetric2InternalConfig(&metric))
	}
	return confs
}

// ConvertApiMetric2InternalConfig
func (c *MetricContext) ConvertApiMetric2InternalConfig(metric *predictionapi.PredictionMetric) *predconf.Config {
	return &predconf.Config{
		DSP:        metric.Algorithm.DSP,
		Percentile: metric.Algorithm.Percentile,
	}
}
