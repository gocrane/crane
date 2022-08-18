package metricnaming

import (
	"github.com/gocrane/crane/pkg/metricquery"
	"github.com/gocrane/crane/pkg/querybuilder"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// MetricNamer is an interface. it is the bridge between predictor and different data sources and other component such as caller.
type MetricNamer interface {
	// Used for datasource provider, data source provider call QueryBuilder
	QueryBuilder() querybuilder.QueryBuilder
	// Used for predictor now
	BuildUniqueKey() string

	Validate() error

	// Means the caller of this MetricNamer, different caller maybe use the same metric
	Caller() string
}

var _ MetricNamer = &GeneralMetricNamer{}

type GeneralMetricNamer struct {
	Metric     *metricquery.Metric
	CallerName string
}

func (gmn *GeneralMetricNamer) Caller() string {
	return gmn.CallerName
}

func (gmn *GeneralMetricNamer) QueryBuilder() querybuilder.QueryBuilder {
	return NewQueryBuilder(gmn.Metric)
}

func (gmn *GeneralMetricNamer) BuildUniqueKey() string {
	return gmn.CallerName + "/" + gmn.Metric.BuildUniqueKey()
}

func (gmn *GeneralMetricNamer) Validate() error {
	return gmn.Metric.ValidateMetric()
}

type queryBuilderFactory struct {
	metric *metricquery.Metric
}

func (q queryBuilderFactory) Builder(source metricquery.MetricSource) querybuilder.Builder {
	initFunc := querybuilder.GetBuilderFactory(source)
	return initFunc(q.metric)
}

func NewQueryBuilder(metric *metricquery.Metric) querybuilder.QueryBuilder {
	return &queryBuilderFactory{
		metric: metric,
	}
}

func ResourceToWorkloadMetricNamer(target *corev1.ObjectReference, resourceName *corev1.ResourceName, workloadLabelSelector labels.Selector, caller string) MetricNamer {
	// workload
	return &GeneralMetricNamer{
		CallerName: caller,
		Metric: &metricquery.Metric{
			Type:       metricquery.WorkloadMetricType,
			MetricName: resourceName.String(),
			Workload: &metricquery.WorkloadNamerInfo{
				Namespace:  target.Namespace,
				Kind:       target.Kind,
				APIVersion: target.APIVersion,
				Name:       target.Name,
				Selector:   workloadLabelSelector,
			},
		},
	}
}

func ResourceToContainerMetricNamer(namespace, apiVersion, workloadKind, workloadName, containerName string, resourceName corev1.ResourceName, caller string) MetricNamer {
	// container
	return &GeneralMetricNamer{
		CallerName: caller,
		Metric: &metricquery.Metric{
			Type:       metricquery.ContainerMetricType,
			MetricName: resourceName.String(),
			Container: &metricquery.ContainerNamerInfo{
				Namespace:    namespace,
				APIVersion:   apiVersion,
				WorkloadKind: workloadKind,
				WorkloadName: workloadName,
				Name:         containerName,
				Selector:     labels.Everything(),
			},
		},
	}
}
