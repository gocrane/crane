package metricserver

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	customapi "k8s.io/metrics/pkg/apis/custom_metrics/v1beta2"
	externalapi "k8s.io/metrics/pkg/apis/external_metrics/v1beta1"
	metricsapi "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	resourceclient "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
	customclient "k8s.io/metrics/pkg/client/custom_metrics"
	externalclient "k8s.io/metrics/pkg/client/external_metrics"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/metricquery"
)

type MetricsClient interface {
	GetMetricValue(metric *metricquery.Metric) ([]*common.TimeSeries, error)
}

func NewCraneMetricsClient(resourceClient resourceclient.MetricsV1beta1Interface, customClient customclient.CustomMetricsClient, externalClient externalclient.ExternalMetricsClient) MetricsClient {
	return &craneMetricsClient{
		&resourceMetricsClient{resourceClient},
		&customMetricsClient{customClient},
		&externalMetricsClient{externalClient},
	}
}

// craneMetricsClient is a client which supports fetching
// metrics from both the resource metrics API, custom metrics API, external metrics API
type craneMetricsClient struct {
	*resourceMetricsClient
	*customMetricsClient
	*externalMetricsClient
}

func (c craneMetricsClient) GetMetricValue(metric *metricquery.Metric) ([]*common.TimeSeries, error) {
	if metric == nil {
		return nil, fmt.Errorf("metric is null")
	}
	switch metric.Type {
	case metricquery.PodMetricType:
		fallthrough
	case metricquery.WorkloadMetricType:
		fallthrough
	case metricquery.ContainerMetricType:
		fallthrough
	case metricquery.NodeMetricType:
		res, timestamp, err := c.GetResourceMetric(metric)
		if err != nil {
			return nil, err
		}
		return convertResourceMetric2TimeSeries(res, timestamp), nil
	case metricquery.PromQLMetricType:
		return nil, fmt.Errorf("metric type %v do not support metric server resource metric now", metric.Type)
	default:
		return nil, fmt.Errorf("unknown metric type %v", metric.Type)
	}
}

func convertResourceMetric2TimeSeries(info ResourceMetricInfo, timestamp time.Time) []*common.TimeSeries {
	var tsList []*common.TimeSeries
	for _, metricValue := range info {
		ts := common.NewTimeSeries()
		ts.Labels = metricValue.Labels
		ts.AppendSample(timestamp.Unix(), metricValue.Value)
		tsList = append(tsList, ts)
	}
	return tsList
}

type resourceMetricsClient struct {
	client resourceclient.MetricsV1beta1Interface
}

type customMetricsClient struct {
	client customclient.CustomMetricsClient
}

type externalMetricsClient struct {
	client externalclient.ExternalMetricsClient
}

// ResourceMetric contains metric value (the metric values are expected to be the metric as a milli-value)
type ResourceMetric struct {
	Timestamp time.Time
	Window    time.Duration
	Value     float64
	// Labels must keep the same with prometheus labels and other data labels
	// todo: we must remove predictor dependency to the labels for single-metric-multi-series scene
	Labels []common.Label
}

// ResourceMetricInfo contains metrics as an array of ResourceMetric
type ResourceMetricInfo []ResourceMetric

func (c *resourceMetricsClient) GetResourceMetric(metric *metricquery.Metric) (ResourceMetricInfo, time.Time, error) {
	switch metric.Type {
	case metricquery.PodMetricType:
		// now pod has no labels for promql
		return c.podMetric(metric)
	case metricquery.WorkloadMetricType:
		// now workload has no labels for promql
		return c.workloadMetric(metric)
	case metricquery.ContainerMetricType:
		// when it is container, we do not use labels now. because we use key `__all__` aggregated only for resource recommendation, so ignore the labels is ok
		return c.containerMetric(metric)
	case metricquery.NodeMetricType:
		// now node has no labels for promql
		return c.nodeMetric(metric)
	case metricquery.PromQLMetricType:
		return nil, time.Time{}, fmt.Errorf("metric type %v do not support metric server resource metric now", metric.Type)
	default:
		return nil, time.Time{}, fmt.Errorf("unknown metric %v", metric.MetricName)
	}
}

// Because resource metrics of metric server in kubernetes is only support pods and nodes now,
// so if we fetch workloads, we must fetch all the pods of the workload by label selector and aggregate it.
func (c *resourceMetricsClient) workloadMetric(metric *metricquery.Metric) (ResourceMetricInfo, time.Time, error) {
	workload := metric.Workload
	if workload == nil {
		return nil, time.Time{}, fmt.Errorf("metric WorkloadNamerInfo is null")
	}

	selector := ""
	if workload.Selector != nil {
		selector = workload.Selector.String()
	}
	// use resourceVersion=0 to avoid traffic for apiserver to etcd
	metrics, err := c.client.PodMetricses(workload.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector, ResourceVersion: "0"})
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("unable to fetch metrics from resource metrics API: %v", err)
	}

	if len(metrics.Items) == 0 {
		return nil, time.Time{}, fmt.Errorf("no metrics returned from resource metrics API")
	}

	res, timestamp := getWorkloadMetrics(v1.ResourceName(metric.MetricName), metrics.Items)
	return res, timestamp, nil
}

// when it is container, we do not use labels now. because we use __all__ aggregated
func (c *resourceMetricsClient) containerMetric(metric *metricquery.Metric) (ResourceMetricInfo, time.Time, error) {
	container := metric.Container
	if container == nil {
		return nil, time.Time{}, fmt.Errorf("metric ContainerNamerInfo is null")
	}

	selector := ""
	if container.Selector != nil {
		selector = container.Selector.String()
	}
	// now if we use workloadName info only, then we should first fetch workload pods by kube client, then use PodMetricses to get pods metrics
	// each metric model's addSample will trigger this two listing.
	// so we give the workload label selector directly to get pod metricses, use resourceVersion=0 to avoid traffic for apiserver to etcd
	podMetrics, err := c.client.PodMetricses(container.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector, ResourceVersion: "0"})
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("unable to fetch metrics from resource metrics API: %v", err)
	}

	if len(podMetrics.Items) == 0 {
		return nil, time.Time{}, fmt.Errorf("no metrics returned from resource metrics API")
	}

	res, timestamp := getContainerMetrics(v1.ResourceName(metric.MetricName), podMetrics.Items, container.Name)
	return res, timestamp, nil
}

func (c *resourceMetricsClient) podMetric(metric *metricquery.Metric) (ResourceMetricInfo, time.Time, error) {
	pod := metric.Pod
	if pod == nil {
		return nil, time.Time{}, fmt.Errorf("metric PodNamerInfo is null")
	}

	podMetrics, err := c.client.PodMetricses(pod.Namespace).Get(context.TODO(), pod.Name, metav1.GetOptions{ResourceVersion: "0"})
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("unable to fetch metrics from resource metrics API: %v", err)
	}

	res, timestamp := getPodMetrics(v1.ResourceName(metric.MetricName), podMetrics)
	return res, timestamp, nil
}

func (c *resourceMetricsClient) nodeMetric(metric *metricquery.Metric) (ResourceMetricInfo, time.Time, error) {
	node := metric.Node
	if node == nil {
		return nil, time.Time{}, fmt.Errorf("metric NodeNamerInfo is null")
	}

	metrics, err := c.client.NodeMetricses().Get(context.TODO(), node.Name, metav1.GetOptions{})
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("unable to fetch metrics from resource metrics API: %v", err)
	}

	res, timestamp := getNodeMetrics(v1.ResourceName(metric.MetricName), metrics)
	return res, timestamp, nil
}

func getContainerMetrics(resource v1.ResourceName, podMetricsList []metricsapi.PodMetrics, container string) (ResourceMetricInfo, time.Time) {
	res := make(ResourceMetricInfo, 0)
	var total int64
	var timestamp metav1.Time
	var window metav1.Duration

	for _, podMetric := range podMetricsList {
		timestamp = podMetric.Timestamp
		window = podMetric.Window
		for _, containerMetric := range podMetric.Containers {
			if containerMetric.Name != container {
				continue
			}
			if usage, ok := containerMetric.Usage[resource]; ok {
				total += usage.MilliValue()
			}
			// now workload has no labels for promql
			res = append(res, ResourceMetric{
				Timestamp: timestamp.Time,
				Window:    window.Duration,
				Value:     float64(total) / 1000.,
				Labels:    []common.Label{},
			})
		}
	}
	return res, timestamp.Time
}

func getPodMetrics(resource v1.ResourceName, podMetric *metricsapi.PodMetrics) (ResourceMetricInfo, time.Time) {
	res := make(ResourceMetricInfo, 0)
	var total int64
	var timestamp metav1.Time
	var window metav1.Duration
	timestamp = podMetric.Timestamp
	window = podMetric.Window
	for _, containerMetric := range podMetric.Containers {
		if usage, ok := containerMetric.Usage[resource]; ok {
			total += usage.MilliValue()
		}
	}
	// now pod has no labels for promql
	res = append(res, ResourceMetric{
		Timestamp: timestamp.Time,
		Window:    window.Duration,
		Value:     float64(total) / 1000.,
		Labels:    []common.Label{},
	})
	return res, timestamp.Time
}

func getWorkloadMetrics(resource v1.ResourceName, podMetrics []metricsapi.PodMetrics) (ResourceMetricInfo, time.Time) {
	res := make(ResourceMetricInfo, 0)
	var total int64
	var timestamp metav1.Time
	var window metav1.Duration
	for _, podMetric := range podMetrics {
		timestamp = podMetric.Timestamp
		window = podMetric.Window
		for _, containerMetric := range podMetric.Containers {
			if usage, ok := containerMetric.Usage[resource]; ok {
				total += usage.MilliValue()
			}
		}
	}
	// now workload has no labels for promql
	res = append(res, ResourceMetric{
		Timestamp: timestamp.Time,
		Window:    window.Duration,
		Value:     float64(total) / 1000.,
		Labels:    []common.Label{},
	})
	return res, timestamp.Time
}

func getNodeMetrics(resource v1.ResourceName, nodeMetric *metricsapi.NodeMetrics) (ResourceMetricInfo, time.Time) {
	res := make(ResourceMetricInfo, 0)
	var total int64
	var timestamp metav1.Time
	var window metav1.Duration
	timestamp = nodeMetric.Timestamp
	window = nodeMetric.Window
	if usage, ok := nodeMetric.Usage[resource]; ok {
		// whatever is cpu or memory, use milli-value, then divided by 1000 to float
		total += usage.MilliValue()
	}
	// now pod has no labels for promql
	res = append(res, ResourceMetric{
		Timestamp: timestamp.Time,
		Window:    window.Duration,
		Value:     float64(total) / 1000.,
		Labels:    []common.Label{},
	})
	return res, timestamp.Time
}

func (cm *customMetricsClient) GetObjectMetric(metric *metricquery.Metric) (*customapi.MetricValue, time.Time, error) {
	if metric.Type != metricquery.PromQLMetricType {
		return nil, time.Time{}, fmt.Errorf("metric type %v do not support metric server external metrics", metric.Type)
	}
	metrics, err := cm.client.NamespacedMetrics(metric.Prom.Namespace).GetForObjects(schema.GroupKind{Kind: "Pod"}, metric.Prom.Selector, metric.MetricName, metric.Prom.Selector)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("unable to fetch metrics from custom metrics API: %v", err)
	}

	if len(metrics.Items) == 0 {
		return nil, time.Time{}, fmt.Errorf("no metrics returned from custom metrics API")
	}

	timestamp := metrics.Items[0].Timestamp.Time
	res := metrics.Items[0]
	return &res, timestamp, nil
}

// GetExternalMetric gets all the values of a given external metric
// that match the specified selector.
func (c *externalMetricsClient) GetExternalMetric(metric *metricquery.Metric) (*externalapi.ExternalMetricValue, time.Time, error) {
	if metric.Type != metricquery.PromQLMetricType {
		return nil, time.Time{}, fmt.Errorf("metric type %v do not support metric server external metrics", metric.Type)
	}
	metrics, err := c.client.NamespacedMetrics(metric.Prom.Namespace).List(metric.MetricName, metric.Prom.Selector)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("unable to fetch metrics from external metrics API: %v", err)
	}

	if len(metrics.Items) == 0 {
		return nil, time.Time{}, fmt.Errorf("no metrics returned from external metrics API")
	}

	timestamp := metrics.Items[0].Timestamp.Time
	res := metrics.Items[0]
	return &res, timestamp, nil
}
