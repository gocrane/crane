package metricrouter

import (
	"context"
	"fmt"
	"k8s.io/client-go/rest"
	"sync"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/metrics/pkg/apis/custom_metrics"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
)

type MetricRouter interface {
	provider.CustomMetricsProvider
	provider.ExternalMetricsProvider
}

type GenericMetricRouter struct {
	mutex sync.RWMutex

	metricRouterConfig MetricRouterConfig
	kubeClient client.Client

	customMetricServices map[provider.CustomMetricInfo]MetricService
	externalMetricServices map[provider.ExternalMetricInfo]MetricService
}

func NewGenericMetricRouter(client client.Client, config *rest.Config, metricRouterConfig MetricRouterConfig) (MetricRouter, error)  {
	metricRouter := &GenericMetricRouter{
		metricRouterConfig: metricRouterConfig,
		kubeClient: client,
	}

	for _, apiService := range metricRouterConfig.ApiServices {
		metricService, err := NewMetricService(apiService.Service.Namespace, apiService.Service.Name, *apiService.Service.Port, config, client.RESTMapper())
		if err != nil {
			return nil, fmt.Errorf("failed to create metric service %s/%s", apiService.Service.Namespace, apiService.Service.Name)
		}

		if apiService.Group == custom_metrics.GroupName {
			customMetricInfos := metricService.ListAllMetrics()
			for _, info := range customMetricInfos {
				metricRouter.addOrUpdateCustomMetricService(metricService, info)
			}
		}

		if apiService.Group == external_metrics.GroupName {
			externalMetricInfos := metricService.ListAllExternalMetrics()
			for _, info := range externalMetricInfos {
				metricRouter.addOrUpdateExternalMetricService(metricService, info)
			}
		}
	}

	return metricRouter, nil
}

func (r *GenericMetricRouter) GetMetricByName(ctx context.Context, name types.NamespacedName, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValue, error) {
	metricService := r.routerServiceByCustomMetric(info)
	if metricService == nil {
		return nil, fmt.Errorf("failed to router custom metric service, metric info: %s", info.Metric)
	}
	return metricService.GetMetricByName(ctx, name, info, metricSelector)
}

func (r *GenericMetricRouter) GetMetricBySelector(ctx context.Context, namespace string, selector labels.Selector, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValueList, error) {
	metricService := r.routerServiceByCustomMetric(info)
	if metricService == nil {
		return nil, fmt.Errorf("failed to router custom metric service, metric info: %s", info.Metric)
	}
	return metricService.GetMetricBySelector(ctx, namespace, selector, info, metricSelector)
}

func (r *GenericMetricRouter) ListAllMetrics() []provider.CustomMetricInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var customMetrics []provider.CustomMetricInfo
	for info := range r.customMetricServices {
		customMetrics = append(customMetrics, info)
	}

	return customMetrics
}

func (r *GenericMetricRouter) GetExternalMetric(ctx context.Context, namespace string, metricSelector labels.Selector, info provider.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
	metricService := r.routerServiceByExternalMetric(info)
	if metricService == nil {
		return nil, fmt.Errorf("failed to router external metric service, metric info: %s", info.Metric)
	}
	return metricService.GetExternalMetric(ctx, namespace, metricSelector, info)
}

func (r *GenericMetricRouter) ListAllExternalMetrics() []provider.ExternalMetricInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var externalMetrics []provider.ExternalMetricInfo
	for info := range r.externalMetricServices {
		externalMetrics = append(externalMetrics, info)
	}

	return externalMetrics
}

func (r *GenericMetricRouter) routerServiceByCustomMetric(info provider.CustomMetricInfo) MetricService {
	return r.customMetricServices[info]
}

func (r *GenericMetricRouter) routerServiceByExternalMetric(info provider.ExternalMetricInfo) MetricService {
	return r.externalMetricServices[info]

}

func (r *GenericMetricRouter) addOrUpdateCustomMetricService(metricService MetricService, info provider.CustomMetricInfo) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.customMetricServices[info] = metricService
}

func (r *GenericMetricRouter) addOrUpdateExternalMetricService(metricService MetricService, info provider.ExternalMetricInfo) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.externalMetricServices[info] = metricService
}