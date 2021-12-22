package metrics

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
)

type TspMetricCollector struct {
	client.Client
	Logger            logr.Logger
	resourceCpuMetric *prometheus.Desc
	resourceMemMetric *prometheus.Desc
}

func NewTspMetricCollector(client client.Client) *TspMetricCollector {
	return &TspMetricCollector{
		Client: client,
		resourceCpuMetric: prometheus.NewDesc(
			prometheus.BuildFQName("crane", "prediction", "time_series_prediction_resource_cpu"),
			"prediction resource cpu value for TimeSeriesPrediction",
			[]string{"targetKind", "targetName", "targetNamespace", "resourceIdentifier", "type", "resourceQuery", "metricQuery", "expressionQuery", "algorithm", "aggregateKey"},
			nil,
		),
		resourceMemMetric: prometheus.NewDesc(
			prometheus.BuildFQName("crane", "prediction", "time_series_prediction_resource_memory"),
			"prediction resource memory value for TimeSeriesPrediction",
			[]string{"targetKind", "targetName", "targetNamespace", "resourceIdentifier", "type", "resourceQuery", "metricQuery", "expressionQuery", "algorithm", "aggregateKey"},
			nil,
		),
	}
}

// Why Implement prometheus collector ？
// Because the time series prediction timestamp is future timestamp, this way can push timestamp to prometheus
// if use prometheus metric instrument by default, prometheus scrape will use its own scrape timestamp, so that the prediction time series maybe has wrong timestamps in prom.
func (c *TspMetricCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.resourceCpuMetric
	ch <- c.resourceMemMetric
}

func (c *TspMetricCollector) Collect(ch chan<- prometheus.Metric) {
	tspList := &predictionapi.TimeSeriesPredictionList{}
	err := c.List(context.TODO(), tspList)
	if err != nil {
		c.Logger.Error(err, "Collect metrics failed")
		return
	}
	for _, tsp := range tspList.Items {
		metricList := c.getMetrics(&tsp)
		for _, metric := range metricList {
			ch <- metric
		}
	}
}

func (c *TspMetricCollector) getMetrics(tsp *predictionapi.TimeSeriesPrediction) []prometheus.Metric {
	var ms []prometheus.Metric
	pmMap := map[string]predictionapi.PredictionMetric{}
	for _, pm := range tsp.Spec.PredictionMetrics {
		pmMap[pm.ResourceIdentifier] = pm
	}

	//* collected metric "crane_prediction_time_series_prediction_metric" { label:<name:"aggregateKey" value:"nodes-mem#instance=192.168.56.166:9100" > label:<name:"algorithm" value:"percentile" > label:<name:"expressionQuery" value:"" > label:<name:"rawQuery" value:"sum(node_memory_MemTotal_bytes{} - node_memory_MemAvailable_bytes{}) by (instance)" > label:<name:"resourceIdentifier" value:"nodes-mem" > label:<name:"resourceQuery" value:"" > label:<name:"targetKind" value:"Node" > label:<name:"targetName" value:"192.168.56.166" > label:<name:"targetNamespace" value:"" > label:<name:"type" value:"RawQuery" > gauge:<value:1.82784510645e+06 > timestamp_ms:1639466220000 } was collected before with the same name and label values

	for _, metricStatus := range tsp.Status.PredictionMetrics {
		outputMetrics := c.computePredictionMetric(tsp, pmMap, metricStatus)
		ms = append(ms, outputMetrics...)
	}
	return ms
}

func (c *TspMetricCollector) computePredictionMetric(tsp *predictionapi.TimeSeriesPrediction, pmMap map[string]predictionapi.PredictionMetric, status predictionapi.PredictionMetricStatus) []prometheus.Metric {
	var ms []prometheus.Metric
	now := time.Now().Unix()
	metricConf := pmMap[status.ResourceIdentifier]
	resourceQuery := ""
	metricQuery := ""
	expressionQuery := ""
	if metricConf.ResourceQuery != nil {
		resourceQuery = metricConf.ResourceQuery.String()
	}
	if metricConf.MetricQuery != nil {
		metricQuery = metricSelectorToQueryExpr(metricConf.MetricQuery)
	}

	if metricConf.ExpressionQuery != nil {
		expressionQuery = metricConf.ExpressionQuery.Expression
	}

	for _, data := range status.Prediction {
		key := AggregateSignalKey(status.ResourceIdentifier, data.Labels)
		labelValues := []string{
			tsp.Spec.TargetRef.Kind,
			tsp.Spec.TargetRef.Name,
			tsp.Spec.TargetRef.Namespace,
			status.ResourceIdentifier,
			string(metricConf.Type),
			resourceQuery,
			metricQuery,
			expressionQuery,
			string(metricConf.Algorithm.AlgorithmType),
			key,
		}
		samples := data.Samples
		sort.Slice(samples, func(i, j int) bool {
			if samples[i].Timestamp < samples[j].Timestamp {
				return true
			} else {
				return false
			}
		})

		// just one timestamp point, because prometheus collector will hash the label values, same label values is not valid
		for _, sample := range samples {
			if sample.Timestamp >= now {
				ts := time.Unix(sample.Timestamp, 0)
				value, err := strconv.ParseFloat(sample.Value, 64)
				if err != nil {
					c.Logger.Error(err, "Failed to parse sample value", "value", value)
					continue
				}
				// only collect resource query cpu or memory now.
				if resourceQuery == v1.ResourceCPU.String() {
					s := prometheus.NewMetricWithTimestamp(ts, prometheus.MustNewConstMetric(c.resourceCpuMetric, prometheus.GaugeValue, value, labelValues...))
					ms = append(ms, s)
				} else if resourceQuery == v1.ResourceMemory.String() {
					s := prometheus.NewMetricWithTimestamp(ts, prometheus.MustNewConstMetric(c.resourceMemMetric, prometheus.GaugeValue, value, labelValues...))
					ms = append(ms, s)
				}
				break
			}
		}
	}
	return ms
}

func AggregateSignalKey(id string, labels []predictionapi.Label) string {
	labelSet := make([]string, 0, len(labels)+1)
	for _, label := range labels {
		labelSet = append(labelSet, label.Name+"="+label.Value)
	}
	sort.Strings(labelSet)
	return id + "#" + strings.Join(labelSet, ",")
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
