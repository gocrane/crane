package autoscaling

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/custom_metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/known"
)

type metricValue struct {
	timestamp int64
	value     float64
}

var _ provider.CustomMetricsProvider = &MetricProvider{}

// MetricProvider is a implementation of provider.MetricsProvider which provide predictive metric for resource
type MetricProvider struct {
	client   client.Client
	recorder record.EventRecorder
}

// NewMetricProvider returns an instance of metricProvider
func NewMetricProvider(client client.Client, recorder record.EventRecorder) provider.CustomMetricsProvider {
	provider := &MetricProvider{
		client:   client,
		recorder: recorder,
	}
	return provider
}

func (p *MetricProvider) GetMetricByName(ctx context.Context, name types.NamespacedName, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValue, error) {
	return nil, apiErrors.NewServiceUnavailable("not supported")
}

// GetMetricBySelector fetches metric for pod resources, get predictive metric from giving selector
func (p *MetricProvider) GetMetricBySelector(ctx context.Context, namespace string, selector labels.Selector, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValueList, error) {
	klog.Info("Get metric for custom metric", "GroupResource", info.GroupResource.String(), "namespace", namespace, "metric", info.Metric, "selector", selector.String(), "metricSelector", metricSelector.String())

	var matchingMetrics []custom_metrics.MetricValue
	prediction, err := p.GetPodPrediction(ctx, namespace, metricSelector)
	if err != nil {
		return nil, err
	}

	pods, err := p.GetPods(ctx, namespace, selector)
	if err != nil {
		return nil, err
	}

	readyPods := GetReadyPods(pods)
	if len(readyPods) == 0 {
		return nil, fmt.Errorf("Failed to get ready pods. ")
	}

	isPredicting := false
	// check prediction is ongoing
	if prediction.Status.Conditions != nil {
		for _, condition := range prediction.Status.Conditions {
			if condition.Type == predictionapi.PredictionConditionPredicting && condition.Status == metav1.ConditionTrue {
				isPredicting = true
			}
		}
	}

	if !isPredicting {
		return nil, fmt.Errorf("PodGroupPrediction is not predicting. ")
	}

	timeSeries := prediction.Status.Aggregation[info.Metric]
	// check time series for current metric is fulfilled
	if timeSeries == nil {
		return nil, fmt.Errorf("TimeSeries is empty, metric name %s", info.Metric)
	}

	// get the largest value from timeSeries
	// use the largest value will bring up the scaling up and defer the scaling down
	timestampStart := time.Now()
	timestamepEnd := timestampStart.Add(prediction.Spec.PredictionWindow.Duration)
	largestMetricValue := &metricValue{}
	for _, v := range timeSeries {
		// exclude values that not in time range
		if v.Timestamp < timestampStart.Unix() || v.Timestamp > timestamepEnd.Unix() {
			continue
		}

		valueFloat, err := strconv.ParseFloat(v.Value, 32)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse value to float: %v ", err)
		}
		if valueFloat > largestMetricValue.value {
			largestMetricValue.value = valueFloat
			largestMetricValue.timestamp = v.Timestamp
		}
	}

	averageValue := int64(math.Round(largestMetricValue.value * 1000 / float64(len(readyPods))))

	klog.Info("Provide custom metric ", "metric", info.Metric, "average value", float64(averageValue)/1000)

	for name := range readyPods {
		metric := custom_metrics.MetricValue{
			DescribedObject: custom_metrics.ObjectReference{
				APIVersion: "v1",
				Kind:       "Pod",
				Name:       name,
				Namespace:  namespace,
			},
			Metric: custom_metrics.MetricIdentifier{
				Name: info.Metric,
			},
			Timestamp: metav1.Now(),
		}

		if info.Metric == known.MetricNamePodCpuUsage {
			metric.Value = *resource.NewMilliQuantity(averageValue, resource.DecimalSI)
		} else if info.Metric == known.MetricNamePodMemoryUsage {
			metric.Value = *resource.NewMilliQuantity(averageValue, resource.BinarySI)
		}

		matchingMetrics = append(matchingMetrics, metric)
	}

	return &custom_metrics.MetricValueList{
		Items: matchingMetrics,
	}, nil
}

// ListAllMetrics returns all available custom metrics.
func (p *MetricProvider) ListAllMetrics() []provider.CustomMetricInfo {
	klog.Info("List all custom metrics")

	return []provider.CustomMetricInfo{
		{
			GroupResource: schema.GroupResource{Group: "", Resource: "pods"},
			Namespaced:    true,
			Metric:        known.MetricNamePodCpuUsage,
		},
		{
			GroupResource: schema.GroupResource{Group: "", Resource: "pods"},
			Namespaced:    true,
			Metric:        known.MetricNamePodMemoryUsage,
		},
	}
}

func (p *MetricProvider) GetPodPrediction(ctx context.Context, namespace string, metricSelector labels.Selector) (*predictionapi.PodGroupPrediction, error) {
	labelSelector, err := labels.ConvertSelectorToLabelsMap(metricSelector.String())
	if err != nil {
		klog.Error(err, "Failed to convert metric selectors to labels")
		return nil, err
	}

	matchingLabels := client.MatchingLabels(map[string]string{"app.kubernetes.io/managed-by": known.EffectiveHorizontalPodAutoscalerManagedBy})
	// merge metric selectors
	for key, value := range labelSelector {
		matchingLabels[key] = value
	}

	predictionList := &predictionapi.PodGroupPredictionList{}
	opts := []client.ListOption{
		matchingLabels,
		client.InNamespace(namespace),
	}
	err = p.client.List(ctx, predictionList, opts...)
	if err != nil {
		return nil, fmt.Errorf("Failed to get PodGroupPrediction when get custom metric ")
	} else if len(predictionList.Items) != 1 {
		return nil, fmt.Errorf("Only one PodGroupPrediction should match the selector %s ", metricSelector.String())
	}

	return &predictionList.Items[0], nil
}

func (p *MetricProvider) GetPods(ctx context.Context, namespace string, selector labels.Selector) ([]v1.Pod, error) {
	labelSelector, err := labels.ConvertSelectorToLabelsMap(selector.String())
	if err != nil {
		klog.Error(err, "Failed to convert selectors to labels")
		return nil, err
	}

	podList := &v1.PodList{}
	opts := []client.ListOption{
		client.MatchingLabels(labelSelector),
		client.InNamespace(namespace),
	}
	err = p.client.List(ctx, podList, opts...)
	if err != nil || len(podList.Items) == 0 {
		return nil, fmt.Errorf("Failed to get pods when get custom metric ")
	}

	return podList.Items, nil
}

// GetReadyPods return a set with ready pod names
func GetReadyPods(pods []v1.Pod) sets.String {
	readyPods := sets.String{}

	for _, pod := range pods {
		if pod.DeletionTimestamp != nil || pod.Status.Phase != v1.PodRunning {
			continue
		}
		readyPods.Insert(pod.Name)
	}
	return readyPods
}
