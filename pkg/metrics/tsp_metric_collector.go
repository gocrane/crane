package metrics

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
)

type TspMetricCollector struct {
	client.Client
	resourceMetric           *prometheus.Desc
	externalMetric           *prometheus.Desc
	resourceMetricWithWindow *prometheus.Desc
	externalMetricWithWindow *prometheus.Desc
}

func NewTspMetricCollector(client client.Client) *TspMetricCollector {
	return &TspMetricCollector{
		Client: client,
		resourceMetric: prometheus.NewDesc(
			prometheus.BuildFQName("crane", "prediction", "time_series_prediction_resource"),
			"prediction resource value for TimeSeriesPrediction",
			[]string{"targetKind", "targetName", "targetNamespace", "resourceIdentifier", "type", "algorithm", "resource"},
			nil,
		),
		resourceMetricWithWindow: prometheus.NewDesc(
			prometheus.BuildFQName("crane", "prediction", "time_series_prediction_resource_with_window"),
			"prediction resource value for TimeSeriesPrediction with predictionWindow",
			[]string{"targetKind", "targetName", "targetNamespace", "resourceIdentifier", "type", "algorithm", "resource"},
			nil,
		),
		externalMetric: prometheus.NewDesc(
			prometheus.BuildFQName("crane", "prediction", "time_series_prediction_external"),
			"prediction external value for TimeSeriesPrediction",
			[]string{"targetKind", "targetName", "targetNamespace", "resourceIdentifier", "type", "algorithm"},
			nil,
		),
		externalMetricWithWindow: prometheus.NewDesc(
			prometheus.BuildFQName("crane", "prediction", "time_series_prediction_external_with_window"),
			"prediction external value for TimeSeriesPrediction with predictionWindow",
			[]string{"targetKind", "targetName", "targetNamespace", "resourceIdentifier", "type", "algorithm"},
			nil,
		),
	}
}

// Why Implement prometheus collector ？
// Because the time series prediction timestamp is future timestamp, this way can push timestamp to prometheus
// if use prometheus metric instrument by default, prometheus scrape will use its own scrape timestamp, so that the prediction time series maybe has wrong timestamps in prom.
func (c *TspMetricCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.resourceMetric
	ch <- c.resourceMetricWithWindow
	ch <- c.externalMetric
	ch <- c.externalMetricWithWindow
}

func (c *TspMetricCollector) Collect(ch chan<- prometheus.Metric) {
	tspList := &predictionapi.TimeSeriesPredictionList{}
	err := c.List(context.TODO(), tspList)
	if err != nil {
		klog.Error(err, "Collect metrics failed")
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

	for _, data := range status.Prediction {
		labelValues := []string{
			tsp.Spec.TargetRef.Kind,
			tsp.Spec.TargetRef.Name,
			tsp.Spec.TargetRef.Namespace,
			status.ResourceIdentifier,
			string(metricConf.Type),
			string(metricConf.Algorithm.AlgorithmType),
		}

		if metricConf.Type == "ResourceQuery" {
			labelValues = append(labelValues, metricConf.ResourceQuery.String())
		}

		samples := data.Samples
		sort.Slice(samples, func(i, j int) bool {
			return samples[i].Timestamp < samples[j].Timestamp
		})

		// just one timestamp point, because prometheus collector will hash the label values, same label values is not valid
		for _, v := range samples {
			if v.Timestamp >= now {
				ts := time.Unix(v.Timestamp, 0)
				value, err := strconv.ParseFloat(v.Value, 64)
				if err != nil {
					klog.Error(err, "Failed to parse sample value", "value", value)
					continue
				}
				//collect resource metric cpu or memory.
				if metricConf.ResourceQuery != nil {
					s := prometheus.NewMetricWithTimestamp(ts, prometheus.MustNewConstMetric(c.resourceMetric, prometheus.GaugeValue, value, labelValues...))
					ms = append(ms, s)
				}
				//collect external metric.
				if metricConf.ExpressionQuery != nil {
					s := prometheus.NewMetricWithTimestamp(ts, prometheus.MustNewConstMetric(c.externalMetric, prometheus.GaugeValue, value, labelValues...))
					ms = append(ms, s)
				}
				break
			}
		}

		// get the largest value from timeSeries
		// use the largest value will bring up the scaling up and defer the scaling down
		timestampStart := time.Now()
		timestampEnd := timestampStart.Add(time.Duration(tsp.Spec.PredictionWindowSeconds) * time.Second)
		var metricValue float64

		hasValidSample := false
		//hpa metrics
		for _, v := range samples {
			if v.Timestamp < timestampStart.Unix() || v.Timestamp > timestampEnd.Unix() {
				continue
			}

			valueFloat, err := strconv.ParseFloat(v.Value, 32)
			if err != nil {
				klog.Error(err, "Failed to parse sample value", "value", v.Value)
				continue
			}

			if valueFloat > metricValue {
				hasValidSample = true
				metricValue = valueFloat
			}
		}

		if !hasValidSample {
			klog.Error("TimeSeries is outdated, ResourceIdentifier name %s", status.ResourceIdentifier)
			return ms
		}

		//collect resource metric cpu or memory.
		if metricConf.ResourceQuery != nil {
			s := prometheus.NewMetricWithTimestamp(timestampStart, prometheus.MustNewConstMetric(c.resourceMetricWithWindow, prometheus.GaugeValue, metricValue, labelValues...))
			ms = append(ms, s)
		}
		//collect external metric.
		if metricConf.ExpressionQuery != nil {
			s := prometheus.NewMetricWithTimestamp(timestampStart, prometheus.MustNewConstMetric(c.externalMetricWithWindow, prometheus.GaugeValue, metricValue, labelValues...))
			ms = append(ms, s)
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
