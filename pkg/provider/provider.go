package provider

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"

	predictionapi "github.com/gocrane-io/api/prediction/v1alpha1"
)

type metricValue struct {
	timestamp int64
	value     float64
}

var _ provider.ExternalMetricsProvider = &metricProvider{}

// metricProvider is a implementation of provider.MetricsProvider which provide predictive metric for resource
type metricProvider struct {
	client   client.Client
	recorder record.EventRecorder
	logger   logr.Logger
}

// NewMetricProvider returns an instance of metricProvider
func NewMetricProvider(logger logr.Logger, client client.Client, recorder record.EventRecorder) provider.ExternalMetricsProvider {
	provider := &metricProvider{
		logger:   logger.WithName("metric-provider"),
		client:   client,
		recorder: recorder,
	}
	return provider
}

func (p *metricProvider) GetExternalMetric(ctx context.Context, namespace string, metricSelector labels.Selector, info provider.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
	p.logger.Info("Get external metric", "namespace", namespace, "labelSelector", metricSelector.String(), "metric", info.Metric)

	var matchingMetrics []external_metrics.ExternalMetricValue
	labelSelector, err := labels.ConvertSelectorToLabelsMap(metricSelector.String())
	if err != nil {
		p.logger.Error(err, "Failed to convert metric selectors to labels")
		return nil, err
	}

	matchingLabels := client.MatchingLabels(map[string]string{"app.kubernetes.io/managed-by": "ahpa-operator"})
	// merge metric selectors
	for key, value := range labelSelector {
		matchingLabels[key] = value
	}

	predictionList := &predictionapi.PodGroupPredictionList{}
	// todo: handle namespace
	opts := []client.ListOption{
		matchingLabels,
	}
	err = p.client.List(ctx, predictionList, opts...)
	if err != nil {
		return nil, fmt.Errorf("Failed to get PodGroupPrediction when get external metric ")
	} else if len(predictionList.Items) != 1 {
		return nil, fmt.Errorf("Only one PodGroupPrediction should match the selector %s ", metricSelector.String())
	}

	prediction := predictionList.Items[0]
	// check prediction is ongoing
	if prediction.Status.Status != predictionapi.PredictionStatusPredicting {
		return nil, fmt.Errorf("PodGroupPrediction is not ready, current status %s ", prediction.Status.Status)
	}

	timeSeries := prediction.Status.Aggregation[info.Metric]
	// check time series for current metric is fulfilled
	if timeSeries == nil {
		return nil, fmt.Errorf("TimeSeries is empty, metric name %s", info.Metric)
	}

	// get the largest value in the future
	largestMetricValue := &metricValue{}
	for _, v := range timeSeries {
		valueFloat, err := strconv.ParseFloat(v.Value, 32)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse value to float: %v ", err)
		}
		if valueFloat > largestMetricValue.value {
			largestMetricValue.value = valueFloat
			largestMetricValue.timestamp = v.Timestamp
		}
	}

	p.logger.Info("Provide external metric", "metric", info.Metric, "value", largestMetricValue.value)

	metric := external_metrics.ExternalMetricValue{
		MetricName: info.Metric,
		Value:      *resource.NewQuantity(int64(math.Round(largestMetricValue.value)), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}
	matchingMetrics = append(matchingMetrics, metric)

	return &external_metrics.ExternalMetricValueList{
		Items: matchingMetrics,
	}, nil
}

func (p *metricProvider) ListAllExternalMetrics() []provider.ExternalMetricInfo {
	externalMetricsInfo := []provider.ExternalMetricInfo{}

	predictionList := &predictionapi.PodGroupPredictionList{}
	opts := []client.ListOption{
		client.MatchingLabels(map[string]string{"app.kubernetes.io/managed-by": "ahpa-operator"}),
	}
	err := p.client.List(context.TODO(), predictionList, opts...)
	if err != nil {
		p.logger.Error(err, "Failed to list PodGroupPrediction")
		return nil
	}

	metricSet := sets.String{}
	for _, prediction := range predictionList.Items {
		for _, metricConfig := range prediction.Spec.MetricPredictionConfigs {
			metricSet.Insert(metricConfig.MetricName)
		}
	}

	for metric := range metricSet {
		externalMetricsInfo = append(externalMetricsInfo, provider.ExternalMetricInfo{Metric: metric})
	}
	return externalMetricsInfo
}
