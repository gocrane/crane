package metricserver

import (
	cacheddiscovery "k8s.io/client-go/discovery/cached"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/klog/v2"
	resourceclient "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
	customclient "k8s.io/metrics/pkg/client/custom_metrics"
	externalclient "k8s.io/metrics/pkg/client/external_metrics"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/metricquery"
	"github.com/gocrane/crane/pkg/providers"
)

var _ providers.RealTime = &metricsServer{}

// ??? do we need to cache all resource metrics to avoid traffic to apiserver. because vpa to apiserver call is triggered by time tick to list all metrics periodically,
// it can be controlled by a unified loop. but crane to apiserver call is triggered by each metric prediction query, the traffic can not be controlled universally.
// maybe we can use clients rate limiter.
type metricsServer struct {
	client MetricsClient
}

func (m *metricsServer) QueryLatestTimeSeries(metricNamer metricnaming.MetricNamer) ([]*common.TimeSeries, error) {
	msBuilder := metricNamer.QueryBuilder().Builder(metricquery.MetricServerMetricSource)
	msQuery, err := msBuilder.BuildQuery()
	if err != nil {
		klog.Errorf("Failed to QueryLatestTimeSeries metricNamer %v, err: %v", metricNamer.BuildUniqueKey(), err)
		return nil, err
	}
	klog.V(6).Infof("QueryLatestTimeSeries metricNamer %v", metricNamer.BuildUniqueKey())
	return m.client.GetMetricValue(msQuery.GenericQuery.Metric)
}

func NewProvider(config *rest.Config) (providers.RealTime, error) {
	rootClient, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	// Use a discovery client capable of being refreshed.
	discoveryClientSet := rootClient.Discovery()
	cachedClient := cacheddiscovery.NewMemCacheClient(discoveryClientSet)
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedClient)
	apiVersionsGetter := customclient.NewAvailableAPIsGetter(discoveryClientSet)

	resourceClient, err := resourceclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	customClient := customclient.NewForConfig(config, restMapper, apiVersionsGetter)

	externalClient, err := externalclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &metricsServer{
		client: NewCraneMetricsClient(resourceClient, customClient, externalClient),
	}, nil
}
