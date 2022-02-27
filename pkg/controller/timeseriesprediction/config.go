package timeseriesprediction

import (
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/prediction"
	predconf "github.com/gocrane/crane/pkg/prediction/config"
)

type MetricContext struct {
	Namespace        string
	TargetKind       string
	Name             string
	SeriesPrediction *predictionapi.TimeSeriesPrediction
	predictors       map[predictionapi.AlgorithmType]prediction.Interface
}

func NewMetricContext(seriesPrediction *predictionapi.TimeSeriesPrediction, predictors map[predictionapi.AlgorithmType]prediction.Interface) *MetricContext {
	c := &MetricContext{
		Namespace:        seriesPrediction.Namespace,
		TargetKind:       seriesPrediction.Spec.TargetRef.Kind,
		Name:             seriesPrediction.Spec.TargetRef.Name,
		SeriesPrediction: seriesPrediction,
		predictors:       predictors,
	}
	if strings.ToLower(c.TargetKind) != strings.ToLower(predconf.TargetKindNode) && seriesPrediction.Spec.TargetRef.Namespace != "" {
		c.Namespace = seriesPrediction.Spec.TargetRef.Namespace
	}
	return c
}

func (c *MetricContext) GetCaller() string {
	return fmt.Sprintf(callerFormat, klog.KObj(c.SeriesPrediction), c.SeriesPrediction.UID)
}

func (c *MetricContext) GetQueryStr(conf *predictionapi.PredictionMetric) string {
	var queryStr string
	if conf.MetricQuery != nil {
		klog.InfoS("GetQueryStr MetricQuery not supported", "tsp", klog.KObj(c.SeriesPrediction), "metricSelector", metricSelectorToQueryExpr(conf.MetricQuery))
		return ""
	}
	if conf.ExpressionQuery != nil {
		queryStr = conf.ExpressionQuery.Expression
		klog.InfoS("GetQueryStr", "tsp", klog.KObj(c.SeriesPrediction), "queryExpr", conf.ExpressionQuery.Expression)
	}
	if conf.ResourceQuery != nil {
		queryStr = c.ResourceToPromQueryExpr(conf.ResourceQuery)
		klog.InfoS("GetQueryStr", "tsp", klog.KObj(c.SeriesPrediction), "resourceQuery", conf.ResourceQuery)
	}
	return queryStr
}

func (c *MetricContext) WithApiConfig(conf *predictionapi.PredictionMetric) {

	if predictor, ok := c.predictors[conf.Algorithm.AlgorithmType]; ok {
		internalConf := c.ConvertApiMetric2InternalConfig(conf)
		queryStr := c.GetQueryStr(conf)
		err := predictor.WithQuery(c.GetQueryStr(conf), c.GetCaller(), *internalConf)
		if err != nil {
			klog.InfoS("WithApiConfig WithQuery registered failed", "tsp", klog.KObj(c.SeriesPrediction), "queryStr", queryStr)
			return
		}
		klog.InfoS("WithApiConfig WithQuery registered succeed", "tsp", klog.KObj(c.SeriesPrediction), "queryStr", queryStr)
	}
}

func (c *MetricContext) WithApiConfigs(configs []predictionapi.PredictionMetric) {
	for _, conf := range configs {
		c.WithApiConfig(&conf)
	}
}

func (c *MetricContext) DeleteApiConfig(conf *predictionapi.PredictionMetric) {
	queryStr := c.GetQueryStr(conf)
	klog.InfoS("DeleteApiConfig DeleteQuery", "tsp", klog.KObj(c.SeriesPrediction), "queryStr", queryStr)
	if predictor, ok := c.predictors[conf.Algorithm.AlgorithmType]; ok {
		err := predictor.DeleteQuery(queryStr, c.GetCaller())
		if err != nil {
			klog.InfoS("DeleteApiConfig DeleteQuery deleted failed", "tsp", klog.KObj(c.SeriesPrediction), "queryStr", queryStr)
			return
		}
		klog.InfoS("DeleteApiConfig DeleteQuery deleted succeed", "tsp", klog.KObj(c.SeriesPrediction), "queryStr", queryStr)
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

func (c *MetricContext) ResourceToPromQueryExpr(resourceName *corev1.ResourceName) string {
	if strings.ToLower(c.TargetKind) == strings.ToLower(predconf.TargetKindNode) {
		switch *resourceName {
		case corev1.ResourceCPU:
			return fmt.Sprintf(predconf.NodeCpuUsagePromQLFmtStr, c.Name, c.Name, "5m")
		case corev1.ResourceMemory:
			return fmt.Sprintf(predconf.NodeMemUsagePromQLFmtStr, c.Name, c.Name)
		}
	} else {
		switch *resourceName {
		case corev1.ResourceCPU:
			return fmt.Sprintf(predconf.WorkloadCpuUsagePromQLFmtStr, c.Namespace, c.Name, "5m")
		case corev1.ResourceMemory:
			return fmt.Sprintf(predconf.WorkloadMemUsagePromQLFmtStr, c.Namespace, c.Name)
		}
	}
	return ""
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
	// transfer the workload to query
	if metric.ResourceQuery != nil {
		// todo: different data source has different querys.
		expr := &predictionapi.ExpressionQuery{
			Expression: c.ResourceToPromQueryExpr(metric.ResourceQuery),
		}
		return &predconf.Config{
			Expression: expr,
			DSP:        metric.Algorithm.DSP,
			Percentile: metric.Algorithm.Percentile,
		}
	} else {
		return &predconf.Config{
			Metric:     metric.MetricQuery,
			Expression: metric.ExpressionQuery,
			DSP:        metric.Algorithm.DSP,
			Percentile: metric.Algorithm.Percentile,
		}
	}
}
