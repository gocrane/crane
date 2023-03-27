package metrics

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/scale"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	. "github.com/gocrane/crane/pkg/metricprovider"
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

type PredictionMetric struct {
	Desc               *prometheus.Desc
	TargetKind         string
	TargetName         string
	TargetNamespace    string
	ResourceIdentifier string
	Algorithm          string
	MetricValue        float64
	Timestamp          time.Time
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
	var predictionMetrics []PredictionMetric
	for _, ehpa := range ehpaList.Items {
		namespace := ehpa.Namespace
		if ehpa.Spec.Prediction != nil {
			var tsp predictionapi.TimeSeriesPrediction
			tspName := "ehpa-" + ehpa.Name

			err := c.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: tspName}, &tsp)
			if err != nil {
				klog.Errorf("Failed to get tsp: %v", err)
				return
			}
			metricListTsp := c.getMetricsTsp(&tsp)
			for _, metric := range metricListTsp {
				if MetricContains(predictionMetrics, metric) {
					continue
				}

				ch <- prometheus.NewMetricWithTimestamp(metric.Timestamp, prometheus.MustNewConstMetric(metric.Desc, prometheus.GaugeValue, metric.MetricValue, metric.TargetKind, metric.TargetName, metric.TargetNamespace, metric.ResourceIdentifier, metric.Algorithm))
				predictionMetrics = append(predictionMetrics, metric)
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

func (c *CraneMetricCollector) getMetricsTsp(tsp *predictionapi.TimeSeriesPrediction) []PredictionMetric {
	var ms []PredictionMetric
	pmMap := map[string]predictionapi.PredictionMetric{}
	for _, pm := range tsp.Spec.PredictionMetrics {
		pmMap[pm.ResourceIdentifier] = pm
	}

	//* collected metric "crane_autoscaling_prediction/crane_prediction_tsp" for tsp
	for _, metricStatus := range tsp.Status.PredictionMetrics {
		outputMetrics := c.computePredictionMetric(tsp, pmMap, metricStatus)
		ms = append(ms, outputMetrics...)
	}

	return ms
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

func (c *CraneMetricCollector) computePredictionMetric(tsp *predictionapi.TimeSeriesPrediction, pmMap map[string]predictionapi.PredictionMetric, status predictionapi.PredictionMetricStatus) []PredictionMetric {
	var predictionMetrics []PredictionMetric
	now := time.Now().Unix()
	metricConf := pmMap[status.ResourceIdentifier]

	for _, data := range status.Prediction {
		predictionMetric := PredictionMetric{
			TargetKind:         tsp.Spec.TargetRef.Kind,
			TargetName:         tsp.Spec.TargetRef.Name,
			TargetNamespace:    tsp.Spec.TargetRef.Namespace,
			ResourceIdentifier: status.ResourceIdentifier,
			Algorithm:          string(metricConf.Algorithm.AlgorithmType),
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
					klog.ErrorS(err, "Failed to parse sample value", "value", value)
					continue
				}
				//collect model metric of tsp for Prediction
				predictionMetric.Desc = c.metricPredictionTsp
				predictionMetric.MetricValue = value
				predictionMetric.Timestamp = ts
				predictionMetrics = append(predictionMetrics, predictionMetric)
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
				klog.ErrorS(err, "Failed to parse sample value", "value", v.Value)
				continue
			}

			if valueFloat > metricValue {
				hasValidSample = true
				metricValue = valueFloat
			}
		}

		if !hasValidSample {
			klog.Errorf("TimeSeries is outdated, ResourceIdentifier name %s", status.ResourceIdentifier)
			return predictionMetrics
		}

		//collect external metric of prediction for HorizontalPodAutoscaler
		predictionMetric.Desc = c.metricAutoScalingPrediction
		predictionMetric.MetricValue = metricValue
		predictionMetric.Timestamp = timestampStart
		predictionMetrics = append(predictionMetrics, predictionMetric)
	}
	return predictionMetrics
}

func AggregateSignalKey(id string, labels []predictionapi.Label) string {
	labelSet := make([]string, 0, len(labels)+1)
	for _, label := range labels {
		labelSet = append(labelSet, label.Name+"="+label.Value)
	}
	sort.Strings(labelSet)
	return id + "#" + strings.Join(labelSet, ",")
}

func MetricContains(predictionMetrics []PredictionMetric, pm PredictionMetric) bool {
	for _, m := range predictionMetrics {
		if m.Desc == pm.Desc && m.TargetKind == pm.TargetName && m.TargetNamespace == pm.TargetNamespace && m.ResourceIdentifier == pm.ResourceIdentifier && m.Algorithm == pm.Algorithm {
			return true
		}
	}
	return false
}
