package metricprovider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
)

var _ provider.ExternalMetricsProvider = &ExternalMetricProvider{}

// ExternalMetricProvider implements ehpa external metric as external metric provider which now support cron metric
type ExternalMetricProvider struct {
	client   client.Client
	recorder record.EventRecorder
}

// NewExternalMetricProvider returns an instance of ExternalMetricProvider
func NewExternalMetricProvider(client client.Client, recorder record.EventRecorder) *ExternalMetricProvider {
	return &ExternalMetricProvider{
		client:   client,
		recorder: recorder,
	}
}

const (
	// DefaultCronTargetMetricValue is used to construct a default external cron metric targetValue.
	// When the time is not in cron period, then the hpa will compute the replica counts to DefaultCronTargetMetricValue for the cron metric.
	// So the hpa may scale workload to DefaultCronTargetMetricValue. And finally scale replica depends on the HPA min max replica count the user set.
	DefaultCronTargetMetricValue int64 = 1
)

func (p *ExternalMetricProvider) GetExternalMetric(ctx context.Context, namespace string, metricSelector labels.Selector, info provider.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
	var ehpaList autoscalingapi.EffectiveHorizontalPodAutoscalerList
	err := p.client.List(ctx, &ehpaList)
	if err != nil {
		return &external_metrics.ExternalMetricValueList{}, err
	}
	// Find the cron metric scaler
	var cronScaler *CronScaler
outer:
	for i := range ehpaList.Items {
		cronScalers := GetCronScalersForEHPA(&ehpaList.Items[i])
		for k := range cronScalers {
			if cronScalers[k].Name() == info.Metric {
				cronScaler = cronScalers[k]
				break outer
			}
		}
	}
	if cronScaler == nil {
		return &external_metrics.ExternalMetricValueList{}, fmt.Errorf("cron metric %v/%v not found", namespace, info.Metric)
	}
	isActive, err := cronScaler.IsActive(ctx, time.Now())
	if err != nil {
		return nil, err
	}
	replicas := DefaultCronTargetMetricValue
	if isActive {
		replicas = int64(cronScaler.TargetSize())
	}
	value := external_metrics.ExternalMetricValue{
		MetricName: info.Metric,
		Timestamp:  metav1.Now(),
		Value:      *resource.NewQuantity(replicas, resource.DecimalSI),
	}
	return &external_metrics.ExternalMetricValueList{Items: []external_metrics.ExternalMetricValue{value}}, nil
}

// ListAllExternalMetrics return external cron metrics
// Fetch metrics from cache directly to avoid the performance issue for apiserver when the metrics is large, because this api is called frequently.
func (p *ExternalMetricProvider) ListAllExternalMetrics() []provider.ExternalMetricInfo {
	var metricInfos []provider.ExternalMetricInfo
	var ehpaList autoscalingapi.EffectiveHorizontalPodAutoscalerList
	err := p.client.List(context.TODO(), &ehpaList)
	if err != nil {
		klog.Errorf("Failed to list ehpa: %v", err)
		return metricInfos
	}
	for _, ehpa := range ehpaList.Items {
		for _, cronScale := range ehpa.Spec.Crons {
			metricName := EHPACronMetricName(ehpa.Namespace, ehpa.Name, cronScale)
			metricInfos = append(metricInfos, provider.ExternalMetricInfo{Metric: metricName})
		}
	}
	return metricInfos
}

// EHPACronMetricName return the hpa cron external metric name from ehpa cron scale spec
// construct the cron metric name by ehpa namespace, name, cron name, cron timezone, cron start, cron end
// make sure each ehpa cron scale metric name is unique.
func EHPACronMetricName(namespace, name string, cronScale autoscalingapi.CronSpec) string {
	// same timezone return different cases when in different machine. transfer to lower case
	timezone := GetCronScaleLocation(cronScale)
	// metric name must be lower case, can not container upper case: https://github.com/kubernetes/kubernetes/issues/72996
	return NormalizeString(strings.ToLower(fmt.Sprintf("cron-%v-%v-%v-%v-%v-%v", namespace, name, cronScale.Name, strings.ToLower(timezone.String()), shapeCronTimeFormat(cronScale.Start), shapeCronTimeFormat(cronScale.End))))
}

// GetCronScaleLocation return the cronScale location, default is UTC when it is not specified in spec
func GetCronScaleLocation(cronScale autoscalingapi.CronSpec) *time.Location {
	t := time.Now().UTC()
	timezone := t.Location()
	var err error
	if cronScale.TimeZone != nil {
		timezone, err = time.LoadLocation(*cronScale.TimeZone)
		if err != nil {
			klog.Errorf("Failed to parse timezone %v, use default %+v, err: %v", *cronScale.TimeZone, timezone, err)
			timezone = t.Location()
			return timezone
		}
	}
	return timezone
}

func GetCronScalersForEHPA(ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler) []*CronScaler {
	var scalers []*CronScaler
	for _, cronScale := range ehpa.Spec.Crons {
		cronMetricName := EHPACronMetricName(ehpa.Namespace, ehpa.Name, cronScale)
		scalers = append(scalers, NewCronScaler(&CronTrigger{
			Name:     cronMetricName,
			Location: GetCronScaleLocation(cronScale),
			Start:    cronScale.Start,
			End:      cronScale.End,
		}, ehpa, cronScale.TargetReplicas))
	}
	return scalers
}

func shapeCronTimeFormat(s string) string {
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "*", "x")
	s = strings.ReplaceAll(s, "/", "sl")
	s = strings.ReplaceAll(s, "?", "qm")
	return s
}

func NormalizeString(s string) string {
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, ":", "-")
	s = strings.ReplaceAll(s, "%", "-")
	return s
}

type CronScaler struct {
	trigger        *CronTrigger
	ref            *autoscalingapi.EffectiveHorizontalPodAutoscaler
	targetReplicas int32
}

func NewCronScaler(trigger *CronTrigger, ref *autoscalingapi.EffectiveHorizontalPodAutoscaler, targetReplicas int32) *CronScaler {
	return &CronScaler{
		trigger:        trigger,
		ref:            ref,
		targetReplicas: targetReplicas,
	}
}

func (cs *CronScaler) IsActive(ctx context.Context, now time.Time) (bool, error) {
	return cs.trigger.IsActive(ctx, now)
}

func (cs *CronScaler) Name() string {
	return cs.trigger.Name
}

func (cs *CronScaler) TargetSize() int32 {
	return cs.targetReplicas
}
