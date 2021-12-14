package tsp

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/gocrane/api/prediction/v1alpha1"
)

func (c *Controller) RegisterMetric() {
	metrics.Registry.MustRegister(c)
}

// Why Implement prometheus collector ？
// Because the time series prediction timestamp is future timestamp, this way can push timestamp to prometheus
// if use prometheus metric instrument by default, prometheus scrape will use its own scrape timestamp, so that the prediction time series maybe has wrong timestamps in prom.
func (c *Controller) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.metric
}

func (c *Controller) Collect(ch chan<- prometheus.Metric) {
	// your logic should be placed here
	tspList := &v1alpha1.TimeSeriesPredictionList{}
	err := c.List(context.TODO(), tspList)
	if err != nil {
		c.Logger.Error(err, "Collect metrics failed")
		return
	}
	for _, tsp := range tspList.Items {
		metrics := c.getMetrics(&tsp)
		for _, metric := range metrics {
			ch <- metric
		}
	}
}

func (c *Controller) getMetrics(tsp *v1alpha1.TimeSeriesPrediction) []prometheus.Metric {
	var ms []prometheus.Metric
	pmMap := map[string]v1alpha1.PredictionMetric{}
	for _, pm := range tsp.Spec.PredictionMetrics {
		pmMap[pm.ResourceIdentifier] = pm
	}

	//* collected metric "crane_prediction_time_series_prediction_metric" { label:<name:"aggregateKey" value:"nodes-mem#instance=192.168.56.166:9100" > label:<name:"algorithm" value:"percentile" > label:<name:"expressionQuery" value:"" > label:<name:"rawQuery" value:"sum(node_memory_MemTotal_bytes{} - node_memory_MemAvailable_bytes{}) by (instance)" > label:<name:"resourceIdentifier" value:"nodes-mem" > label:<name:"resourceQuery" value:"" > label:<name:"targetKind" value:"Node" > label:<name:"targetName" value:"192.168.56.166" > label:<name:"targetNamespace" value:"" > label:<name:"type" value:"RawQuery" > gauge:<value:1.82784510645e+06 > timestamp_ms:1639466220000 } was collected before with the same name and label values

	for _, metric := range tsp.Status.PredictionMetrics {
		now := time.Now().Unix()
		metricConf := pmMap[metric.ResourceIdentifier]
		resourceQuery := ""
		expressionQuery := ""
		rawQuery := ""
		if metricConf.ResourceQuery != nil {
			resourceQuery = metricConf.ResourceQuery.String()
		}
		if metricConf.ExpressionQuery != nil {
			expressionQuery = metricSelectorToQueryExpr(metricConf.ExpressionQuery)
		}
		if metricConf.RawQuery != nil {
			rawQuery = metricConf.RawQuery.Expression
		}
		for _, data := range metric.Prediction {
			key := AggregateSignalKey(metric.ResourceIdentifier, data.Labels)
			labelValues := []string{
				tsp.Spec.TargetRef.Kind,
				tsp.Spec.TargetRef.Name,
				tsp.Spec.TargetRef.Namespace,
				metric.ResourceIdentifier,
				string(metricConf.Type),
				resourceQuery,
				expressionQuery,
				rawQuery,
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
					s := prometheus.NewMetricWithTimestamp(ts, prometheus.MustNewConstMetric(c.metric, prometheus.GaugeValue, value, labelValues...))
					ms = append(ms, s)
					break
				}
			}
		}
	}
	return ms
}

func AggregateSignalKey(id string, labels []v1alpha1.Label) string {
	labelSet := make([]string, 0, len(labels)+1)
	for _, label := range labels {
		labelSet = append(labelSet, label.Name+"="+label.Value)
	}
	sort.Strings(labelSet)
	return id + "#" + strings.Join(labelSet, ",")
}

func metricSelectorToQueryExpr(m *v1alpha1.ExpressionQuery) string {
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
