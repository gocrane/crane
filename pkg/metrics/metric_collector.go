package metrics

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	. "github.com/gocrane/crane/pkg/metricprovider"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/scale"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CraneMetricCollector struct {
	client.Client
	scaler     scale.ScalesGetter
	restMapper meta.RESTMapper
	//external metrics of cron for hpa
	metricAutoScalingCron *prometheus.Desc
	//external metrics of prediction for hpa
	metricAutoScalingPrediction *prometheus.Desc
	//model metrics of tsp
	metricPredictionTsp *prometheus.Desc
}

func NewCraneMetricCollector(client client.Client, scaleClient scale.ScalesGetter, restMapper meta.RESTMapper) *CraneMetricCollector {
	return &CraneMetricCollector{
		Client:     client,
		scaler:     scaleClient,
		restMapper: restMapper,
		metricAutoScalingCron: prometheus.NewDesc(
			prometheus.BuildFQName("crane", "autoscaling", "cron"),
			"external metrics value of cron for HorizontalPodAutoscaler",
			[]string{"targetKind", "targetName", "targetNamespace", "resourceIdentifier"},
			nil,
		),
		metricAutoScalingPrediction: prometheus.NewDesc(
			prometheus.BuildFQName("crane", "autoscaling", "prediction"),
			"external metrics value of prediction for HorizontalPodAutoscaler",
			[]string{"targetKind", "targetName", "targetNamespace", "resourceIdentifier", "algorithm"},
			nil,
		),
		metricPredictionTsp: prometheus.NewDesc(
			prometheus.BuildFQName("crane", "prediction", "tsp"),
			"model metrics value of tsp for Prediction",
			[]string{"targetKind", "targetName", "targetNamespace", "resourceIdentifier", "algorithm"},
			nil,
		),
	}
}

// Why Implement prometheus collector ？
// Because the time series prediction timestamp is future timestamp, this way can push timestamp to prometheus
// if use prometheus metric instrument by default, prometheus scrape will use its own scrape timestamp, so that the prediction time series maybe has wrong timestamps in prom.
func (c *CraneMetricCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.metricAutoScalingCron
	ch <- c.metricAutoScalingPrediction
	ch <- c.metricPredictionTsp
}

func (c *CraneMetricCollector) Collect(ch chan<- prometheus.Metric) {
	var ehpaList autoscalingapi.EffectiveHorizontalPodAutoscalerList
	err := c.List(context.TODO(), &ehpaList)
	if err != nil {
		klog.Errorf("Failed to list ehpa: %v", err)
	}
	var uniqPredictionMetrics []string
	for _, ehpa := range ehpaList.Items {
		namespace := ehpa.Namespace
		if ehpa.Spec.Prediction != nil {
			var tsp predictionapi.TimeSeriesPrediction
			tspName := "ehpa-" + ehpa.Name

			err := c.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: tspName}, &tsp)
			if err != nil {
				klog.Error("Failed to get tsp: %v", err)
				return
			}
			var metricListTsp []prometheus.Metric
			metricListTsp, uniqPredictionMetrics = c.getMetricsTsp(&tsp, uniqPredictionMetrics)
			for _, metric := range metricListTsp {
				ch <- metric
			}
		}

		if ehpa.Spec.Crons != nil {
			metricCron, err := c.getMetricsCron(&ehpa)
			if err != nil {
				klog.Errorf("Failed to get metricCron: %v", err)
				return
			}

			ch <- metricCron
		}

	}
}

func (c *CraneMetricCollector) getMetricsTsp(tsp *predictionapi.TimeSeriesPrediction, uniqPredictionMetrics []string) ([]prometheus.Metric, []string) {
	var ms []prometheus.Metric
	pmMap := map[string]predictionapi.PredictionMetric{}
	for _, pm := range tsp.Spec.PredictionMetrics {
		pmMap[pm.ResourceIdentifier] = pm
	}

	//* collected metric "crane_prediction_time_series_prediction_metric" { label:<name:"aggregateKey" value:"nodes-mem#instance=192.168.56.166:9100" > label:<name:"algorithm" value:"percentile" > label:<name:"expressionQuery" value:"" > label:<name:"rawQuery" value:"sum(node_memory_MemTotal_bytes{} - node_memory_MemAvailable_bytes{}) by (instance)" > label:<name:"resourceIdentifier" value:"nodes-mem" > label:<name:"resourceQuery" value:"" > label:<name:"targetKind" value:"Node" > label:<name:"targetName" value:"192.168.56.166" > label:<name:"targetNamespace" value:"" > label:<name:"type" value:"RawQuery" > gauge:<value:1.82784510645e+06 > timestamp_ms:1639466220000 } was collected before with the same name and label values
	for _, metricStatus := range tsp.Status.PredictionMetrics {
		var outputMetrics []prometheus.Metric
		outputMetrics, uniqPredictionMetrics = c.computePredictionMetric(tsp, pmMap, metricStatus, uniqPredictionMetrics)
		ms = append(ms, outputMetrics...)
	}

	return ms, uniqPredictionMetrics
}

func (c *CraneMetricCollector) getMetricsCron(ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler) (prometheus.Metric, error) {
	cronScalers := GetCronScalersForEHPA(ehpa)
	var activeScalers []*CronScaler
	var errs []error
	for _, cronScaler := range cronScalers {
		isActive, err := cronScaler.IsActive(context.TODO(), time.Now())
		if err != nil {
			errs = append(errs, err)
		}
		if isActive {
			activeScalers = append(activeScalers, cronScaler)
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("%v", errs)
	}
	// Set default replicas same with minReplicas of ehpa
	replicas := *ehpa.Spec.MinReplicas
	// we use the largest targetReplicas specified in cron spec.
	for _, activeScaler := range activeScalers {
		if activeScaler.TargetSize() >= replicas {
			replicas = activeScaler.TargetSize()
		}
	}

	labelValues := []string{
		ehpa.Spec.ScaleTargetRef.Kind,
		ehpa.Spec.ScaleTargetRef.Name,
		ehpa.Namespace,
		"cron",
	}
	return prometheus.NewMetricWithTimestamp(time.Now(), prometheus.MustNewConstMetric(c.metricAutoScalingCron, prometheus.GaugeValue, float64(replicas), labelValues...)), nil
}

func (c *CraneMetricCollector) computePredictionMetric(tsp *predictionapi.TimeSeriesPrediction, pmMap map[string]predictionapi.PredictionMetric, status predictionapi.PredictionMetricStatus, uniqPredictionMetrics []string) ([]prometheus.Metric, []string) {
	var ms []prometheus.Metric
	now := time.Now().Unix()
	metricConf := pmMap[status.ResourceIdentifier]

	for _, data := range status.Prediction {
		labelValues := []string{
			tsp.Spec.TargetRef.Kind,
			tsp.Spec.TargetRef.Name,
			tsp.Spec.TargetRef.Namespace,
			status.ResourceIdentifier,
			string(metricConf.Algorithm.AlgorithmType),
		}
		var currPredictionMetric = strings.Join(labelValues, ".")
		var duplicateMetric bool
		for _, uniqPredictionMetric := range uniqPredictionMetrics {
			if uniqPredictionMetric == currPredictionMetric {
				duplicateMetric = true
			}
		}

		if duplicateMetric {
			continue
		} else {
			uniqPredictionMetrics = append(uniqPredictionMetrics, currPredictionMetric)
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
				//collect model metric of tsp for Prediction
				s := prometheus.NewMetricWithTimestamp(ts, prometheus.MustNewConstMetric(c.metricPredictionTsp, prometheus.GaugeValue, value, labelValues...))
				ms = append(ms, s)
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
			return ms, uniqPredictionMetrics
		}

		//collect external metric of prediction for HorizontalPodAutoscaler
		s := prometheus.NewMetricWithTimestamp(timestampStart, prometheus.MustNewConstMetric(c.metricAutoScalingPrediction, prometheus.GaugeValue, metricValue, labelValues...))
		ms = append(ms, s)
	}
	return ms, uniqPredictionMetrics
}

func AggregateSignalKey(id string, labels []predictionapi.Label) string {
	labelSet := make([]string, 0, len(labels)+1)
	for _, label := range labels {
		labelSet = append(labelSet, label.Name+"="+label.Value)
	}
	sort.Strings(labelSet)
	return id + "#" + strings.Join(labelSet, ",")
}
