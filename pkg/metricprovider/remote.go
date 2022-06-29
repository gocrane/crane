package metricprovider

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	cacheddiscovery "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/custom_metrics"
	"k8s.io/metrics/pkg/apis/custom_metrics/v1beta2"
	"k8s.io/metrics/pkg/apis/external_metrics"
	externalMetricsAPI "k8s.io/metrics/pkg/apis/external_metrics/v1beta1"
	cmClient "k8s.io/metrics/pkg/client/custom_metrics"
	emClient "k8s.io/metrics/pkg/client/external_metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"

	"github.com/gocrane/crane/pkg/utils"
)

type RemoteAdapter struct {
	metricClient         cmClient.CustomMetricsClient
	externalMetricClient emClient.ExternalMetricsClient
	apiVersionsGetter    cmClient.AvailableAPIsGetter
	discoveryClient      discovery.CachedDiscoveryInterface
	restMapper           meta.RESTMapper
}

func NewRemoteAdapter(namespace string, name string, port int, config *rest.Config, client client.Client) (*RemoteAdapter, error) {
	metricConfig := rest.CopyConfig(config)
	metricConfig.Insecure = true
	metricConfig.CAData = nil
	metricConfig.CAFile = ""
	metricConfig.Host = fmt.Sprintf("https://%s.%s.svc:%d", name, namespace, port)

	discoveryClientSet, err := discovery.NewDiscoveryClientForConfig(metricConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %v ", err)
	}

	apiVersionsGetter := cmClient.NewAvailableAPIsGetter(discoveryClientSet)
	cachedClient := cacheddiscovery.NewMemCacheClient(discoveryClientSet)

	// use actual rest mapper here
	metricClient := cmClient.NewForConfig(metricConfig, client.RESTMapper(), apiVersionsGetter)
	externalMetricsClient, err := emClient.NewForConfig(metricConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create external metrics client: %v", err)
	}

	return &RemoteAdapter{
		metricClient:         metricClient,
		externalMetricClient: externalMetricsClient,
		apiVersionsGetter:    apiVersionsGetter,
		discoveryClient:      cachedClient,
		restMapper:           client.RESTMapper(),
	}, err
}

// ListAllMetrics returns all available custom metrics.
func (r *RemoteAdapter) ListAllMetrics() []provider.CustomMetricInfo {
	klog.Info("List all remote custom metrics")

	version, err := r.apiVersionsGetter.PreferredVersion()
	if err != nil {
		klog.Errorf("Failed to get preferred version: %v ", err)
		return nil
	}
	resources, err := r.discoveryClient.ServerResourcesForGroupVersion(version.String())
	if err != nil {
		klog.Errorf("Failed to get resources for %s: %v", version.String(), err)
		return nil
	}

	var metricInfos []provider.CustomMetricInfo
	for _, r := range resources.APIResources {
		parts := strings.SplitN(r.Name, "/", 2)
		if len(parts) != 2 {
			klog.Warningf("ApiResource name is unexpected %s", r.Name)
			continue
		}
		info := provider.CustomMetricInfo{
			GroupResource: schema.ParseGroupResource(parts[0]),
			Namespaced:    r.Namespaced,
			Metric:        parts[1],
		}
		metricInfos = append(metricInfos, info)
	}

	return metricInfos
}

// GetMetricByName get custom metric from remote adapter
func (r *RemoteAdapter) GetMetricByName(ctx context.Context, name types.NamespacedName, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValue, error) {
	klog.Info("Get remote metric by name")

	kind, err := utils.KindForResource(info.GroupResource.Resource, r.restMapper)
	if err != nil {
		return nil, fmt.Errorf("failed to get kind for resource %s: %v ", info.GroupResource.Resource, err)
	}

	var object *v1beta2.MetricValue
	if info.Namespaced {
		object, err = r.metricClient.NamespacedMetrics(name.Namespace).GetForObject(
			schema.GroupKind{Group: info.GroupResource.Group, Kind: kind},
			name.Name,
			info.Metric,
			metricSelector,
		)
	} else {
		object, err = r.metricClient.RootScopedMetrics().GetForObject(
			schema.GroupKind{Group: info.GroupResource.Group, Kind: kind},
			name.Name,
			info.Metric,
			metricSelector,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get metric by name from remote: %v ", err)
	}

	return &custom_metrics.MetricValue{
		DescribedObject: custom_metrics.ObjectReference{
			Kind:            object.DescribedObject.Kind,
			Namespace:       object.DescribedObject.Namespace,
			Name:            object.DescribedObject.Name,
			APIVersion:      object.DescribedObject.APIVersion,
			ResourceVersion: object.DescribedObject.ResourceVersion,
		},
		Metric: custom_metrics.MetricIdentifier{
			Name:     object.Metric.Name,
			Selector: object.Metric.Selector,
		},
		Timestamp:     object.Timestamp,
		WindowSeconds: object.WindowSeconds,
		Value:         object.Value,
	}, nil
}

// GetMetricBySelector get custom metric from remote
func (r *RemoteAdapter) GetMetricBySelector(ctx context.Context, namespace string, selector labels.Selector, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValueList, error) {
	klog.Info("Get remote metric by selector")

	kind, err := utils.KindForResource(info.GroupResource.Resource, r.restMapper)
	if err != nil {
		return nil, fmt.Errorf("failed to get kind for resource %s: %v ", info.GroupResource.Resource, err)
	}

	var objects *v1beta2.MetricValueList
	if info.Namespaced {
		objects, err = r.metricClient.NamespacedMetrics(namespace).GetForObjects(
			schema.GroupKind{
				Group: info.GroupResource.Group,
				Kind:  kind,
			},
			selector,
			info.Metric,
			metricSelector,
		)
	} else {
		objects, err = r.metricClient.RootScopedMetrics().GetForObjects(
			schema.GroupKind{
				Group: info.GroupResource.Group,
				Kind:  info.GroupResource.Resource,
			},
			selector,
			info.Metric,
			metricSelector,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get metric by selector from remote: %v ", err)
	}
	values := make([]custom_metrics.MetricValue, len(objects.Items))
	for i, v := range objects.Items {
		values[i] = custom_metrics.MetricValue{
			DescribedObject: custom_metrics.ObjectReference{
				Kind:            v.DescribedObject.Kind,
				Namespace:       v.DescribedObject.Namespace,
				Name:            v.DescribedObject.Name,
				APIVersion:      v.DescribedObject.APIVersion,
				ResourceVersion: v.DescribedObject.ResourceVersion,
			},
			Metric: custom_metrics.MetricIdentifier{
				Name:     v.Metric.Name,
				Selector: v.Metric.Selector,
			},
			Timestamp:     v.Timestamp,
			WindowSeconds: v.WindowSeconds,
			Value:         v.Value,
		}
	}
	return &custom_metrics.MetricValueList{
		Items: values,
	}, nil
}

// GetExternalMetric get external metric from remote
func (r *RemoteAdapter) GetExternalMetric(ctx context.Context, namespace string, metricSelector labels.Selector, info provider.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
	klog.Info("Get external metric by selector")

	metricList, err := r.externalMetricClient.NamespacedMetrics(namespace).List(info.Metric, metricSelector)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics for external metric %s/%s: %v", namespace, info.Metric, err)
	}
	returnList := &external_metrics.ExternalMetricValueList{
		Items: make([]external_metrics.ExternalMetricValue, len(metricList.Items)),
	}
	for i, m := range metricList.Items {
		returnList.Items[i] = external_metrics.ExternalMetricValue{
			TypeMeta:      metav1.TypeMeta{Kind: m.Kind, APIVersion: m.APIVersion},
			MetricName:    m.MetricName,
			MetricLabels:  m.MetricLabels,
			Timestamp:     m.Timestamp,
			WindowSeconds: m.WindowSeconds,
			Value:         m.Value,
		}
	}
	return returnList, nil
}

// ListAllExternalMetrics list all external metric from remote
func (r *RemoteAdapter) ListAllExternalMetrics() []provider.ExternalMetricInfo {
	klog.Info("Get external metric by selector")

	var externalMetricInfos []provider.ExternalMetricInfo
	resources, err := r.discoveryClient.ServerResourcesForGroupVersion(externalMetricsAPI.SchemeGroupVersion.String())
	if err != nil {
		klog.Errorf("Failed to get external metric resources for %r: %v", externalMetricsAPI.SchemeGroupVersion, err)
		return nil
	}
	for _, r := range resources.APIResources {
		info := provider.ExternalMetricInfo{
			Metric: r.Name,
		}
		externalMetricInfos = append(externalMetricInfos, info)
	}
	return externalMetricInfos
}
