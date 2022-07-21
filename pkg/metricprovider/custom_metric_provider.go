package metricprovider

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/custom_metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/utils"
)

type metricValue struct {
	timestamp int64
	value     float64
}

var _ provider.CustomMetricsProvider = &CustomMetricProvider{}

// CustomMetricProvider is an implementation of provider.CustomMetricProvider which provide predictive metric for resource
type CustomMetricProvider struct {
	client        client.Client
	remoteAdapter *RemoteAdapter
	recorder      record.EventRecorder
}

// NewCustomMetricProvider returns an instance of CustomMetricProvider
func NewCustomMetricProvider(client client.Client, remoteAdapter *RemoteAdapter, recorder record.EventRecorder) provider.CustomMetricsProvider {
	provider := &CustomMetricProvider{
		client:        client,
		remoteAdapter: remoteAdapter,
		recorder:      recorder,
	}
	return provider
}

func (p *CustomMetricProvider) GetMetricByName(ctx context.Context, name types.NamespacedName, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValue, error) {
	klog.Info(fmt.Sprintf("Get metric by name for custom metric, GroupResource %s namespacedName %s metric %s metricSelector %s", info.GroupResource.String(), name.String(), info.Metric, metricSelector.String()))

	if !IsLocalCustomMetric(info, p.client) {
		if p.remoteAdapter != nil {
			return p.remoteAdapter.GetMetricByName(ctx, name, info, metricSelector)
		} else {
			return nil, apiErrors.NewServiceUnavailable("not supported")
		}
	}

	return nil, apiErrors.NewServiceUnavailable("not supported")
}

// GetMetricBySelector fetches metric for custom resources, get predictive metric from giving selector
func (p *CustomMetricProvider) GetMetricBySelector(ctx context.Context, namespace string, selector labels.Selector, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValueList, error) {
	klog.Info(fmt.Sprintf("Get metric by selector for custom metric, Info %v namespace %s selector %s metricSelector %s", info, namespace, selector.String(), metricSelector.String()))

	if !IsLocalCustomMetric(info, p.client) {
		if p.remoteAdapter != nil {
			return p.remoteAdapter.GetMetricBySelector(ctx, namespace, selector, info, metricSelector)
		} else {
			return nil, apiErrors.NewServiceUnavailable("not supported")
		}
	}

	if strings.HasPrefix(info.Metric, "crane") {
		var matchingMetrics []custom_metrics.MetricValue
		prediction, err := GetPrediction(ctx, p.client, namespace, metricSelector)
		if err != nil {
			return nil, err
		}

		pods, err := p.GetPods(ctx, namespace, selector)
		if err != nil {
			return nil, err
		}

		availablePods := utils.GetAvailablePods(pods)
		if len(availablePods) == 0 {
			return nil, fmt.Errorf("failed to get available pods. ")
		}

		timeSeries, err := utils.GetReadyPredictionMetric(info.Metric, prediction)
		if err != nil {
			return nil, err
		}

		// get the largest value from timeSeries
		// use the largest value will bring up the scaling up and defer the scaling down
		timestampStart := time.Now()
		timestampEnd := timestampStart.Add(time.Duration(prediction.Spec.PredictionWindowSeconds) * time.Second)
		largestMetricValue := &metricValue{}
		hasValidSample := false
		for _, v := range timeSeries.Samples {
			// exclude values that not in time range
			if v.Timestamp < timestampStart.Unix() || v.Timestamp > timestampEnd.Unix() {
				continue
			}

			valueFloat, err := strconv.ParseFloat(v.Value, 32)
			if err != nil {
				return nil, fmt.Errorf("failed to parse value to float: %v ", err)
			}
			if valueFloat > largestMetricValue.value {
				hasValidSample = true
				largestMetricValue.value = valueFloat
				largestMetricValue.timestamp = v.Timestamp
			}
		}

		if !hasValidSample {
			return nil, fmt.Errorf("TimeSeries is outdated, metric name %s", info.Metric)
		}

		if info.GroupResource.String() == "pods" {
			averageValue := int64(math.Round(largestMetricValue.value * 1000 / float64(len(availablePods)))) // use available replicas for object metric

			klog.Infof("Provide pod custom metric %s average value %f.", info.Metric, float64(averageValue)/1000)

			for _, pod := range availablePods {
				metric := custom_metrics.MetricValue{
					DescribedObject: custom_metrics.ObjectReference{
						APIVersion: "v1",
						Kind:       "Pod",
						Name:       pod.Name,
						Namespace:  namespace,
					},
					Metric: custom_metrics.MetricIdentifier{
						Name: info.Metric,
					},
					Timestamp: metav1.Now(),
				}

				metric.Value = *resource.NewMilliQuantity(averageValue, resource.DecimalSI)
				matchingMetrics = append(matchingMetrics, metric)
			}

			return &custom_metrics.MetricValueList{
				Items: matchingMetrics,
			}, nil
		}
	}

	return nil, apiErrors.NewServiceUnavailable("metric not found")
}

// ListAllMetrics returns all available custom metrics.
func (p *CustomMetricProvider) ListAllMetrics() []provider.CustomMetricInfo {
	klog.Info("List all custom metrics")

	metricInfos := ListAllLocalMetrics(p.client)

	if p.remoteAdapter != nil {
		metricInfos = append(metricInfos, p.remoteAdapter.ListAllMetrics()...)
	}

	return metricInfos
}

func ListAllLocalMetrics(client client.Client) []provider.CustomMetricInfo {
	var metricInfos []provider.CustomMetricInfo

	metricInfos = append(metricInfos, provider.CustomMetricInfo{
		GroupResource: schema.GroupResource{Group: "", Resource: "pods"},
		Namespaced:    true,
		Metric:        known.MetricNamePodCpuUsage,
	})

	var hpaList autoscalingv2.HorizontalPodAutoscalerList
	err := client.List(context.TODO(), &hpaList)
	if err != nil {
		klog.Errorf("Failed to list hpa: %v", err)
		return metricInfos
	}
	for _, hpa := range hpaList.Items {
		if !strings.HasPrefix(hpa.Name, "ehpa-") {
			// filter hpa that not created by ehpa
			continue
		}
		for _, metric := range hpa.Spec.Metrics {
			if metric.Type == autoscalingv2.PodsMetricSourceType &&
				metric.Pods != nil &&
				metric.Pods.Metric.Selector != nil &&
				metric.Pods.Metric.Selector.MatchLabels != nil {
				if _, exist := metric.Pods.Metric.Selector.MatchLabels[known.EffectiveHorizontalPodAutoscalerUidLabel]; exist {
					metricInfos = append(metricInfos, provider.CustomMetricInfo{Metric: metric.Pods.Metric.Name, Namespaced: true, GroupResource: schema.GroupResource{Group: "", Resource: "pods"}})
				}
			}
		}
	}

	return metricInfos
}

func IsLocalCustomMetric(metricInfo provider.CustomMetricInfo, client client.Client) bool {
	for _, info := range ListAllLocalMetrics(client) {
		if info.Namespaced == metricInfo.Namespaced &&
			info.Metric == metricInfo.Metric &&
			info.GroupResource.String() == metricInfo.GroupResource.String() {
			return true
		}
	}

	return false
}

func GetPrediction(ctx context.Context, kubeclient client.Client, namespace string, metricSelector labels.Selector) (*predictionapi.TimeSeriesPrediction, error) {
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

	predictionList := &predictionapi.TimeSeriesPredictionList{}
	opts := []client.ListOption{
		matchingLabels,
		client.InNamespace(namespace),
	}
	err = kubeclient.List(ctx, predictionList, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to get TimeSeriesPrediction when get custom metric ")
	} else if len(predictionList.Items) != 1 {
		return nil, fmt.Errorf("only one TimeSeriesPrediction should match the selector %s ", metricSelector.String())
	}

	return &predictionList.Items[0], nil
}

func (p *CustomMetricProvider) GetPods(ctx context.Context, namespace string, selector labels.Selector) ([]v1.Pod, error) {
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
		return nil, fmt.Errorf("failed to get pods when get custom metric ")
	}

	return podList.Items, nil
}
