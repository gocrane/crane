package metricprovider

import (
	"context"
	"fmt"

	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/custom_metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
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

	if p.remoteAdapter != nil {
		return p.remoteAdapter.GetMetricByName(ctx, name, info, metricSelector)
	} else {
		return nil, apiErrors.NewServiceUnavailable("not supported")
	}

	return nil, apiErrors.NewServiceUnavailable("not supported")
}

// GetMetricBySelector fetches metric for custom resources, get predictive metric from giving selector
func (p *CustomMetricProvider) GetMetricBySelector(ctx context.Context, namespace string, selector labels.Selector, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValueList, error) {
	klog.Info(fmt.Sprintf("Get metric by selector for custom metric, Info %v namespace %s selector %s metricSelector %s", info, namespace, selector.String(), metricSelector.String()))

	if p.remoteAdapter != nil {
		return p.remoteAdapter.GetMetricBySelector(ctx, namespace, selector, info, metricSelector)
	} else {
		return nil, apiErrors.NewServiceUnavailable("not supported")
	}

	return nil, apiErrors.NewServiceUnavailable("metric not found")
}

// ListAllMetrics returns all available custom metrics.
func (p *CustomMetricProvider) ListAllMetrics() []provider.CustomMetricInfo {
	klog.Info("List all custom metrics")

	var metricInfos []provider.CustomMetricInfo

	if p.remoteAdapter != nil {
		metricInfos = append(metricInfos, p.remoteAdapter.ListAllMetrics()...)
	}

	return metricInfos
}