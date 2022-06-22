package metricprovider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/utils"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
)

var _ provider.ExternalMetricsProvider = &ExternalMetricProvider{}

// ExternalMetricProvider implements ehpa external metric as external metric provider which now support cron metric
type ExternalMetricProvider struct {
	client        client.Client
	remoteAdapter *RemoteAdapter
	recorder      record.EventRecorder
	scaler        scale.ScalesGetter
	restMapper    meta.RESTMapper
}

// NewExternalMetricProvider returns an instance of ExternalMetricProvider
func NewExternalMetricProvider(client client.Client, remoteAdapter *RemoteAdapter, recorder record.EventRecorder, scaleClient scale.ScalesGetter, restMapper meta.RESTMapper) *ExternalMetricProvider {
	return &ExternalMetricProvider{
		client:        client,
		remoteAdapter: remoteAdapter,
		recorder:      recorder,
		scaler:        scaleClient,
		restMapper:    restMapper,
	}
}

const (
	// DefaultCronTargetMetricValue is used to construct a default external cron metric targetValue.
	// So the hpa may scale workload to DefaultCronTargetMetricValue. And finally scale replica depends on the HPA min max replica count the user set.
	DefaultCronTargetMetricValue int32 = 1
)

// GetExternalMetric each ehpa mapping to only one external cron metric. metric name is ehpa name
func (p *ExternalMetricProvider) GetExternalMetric(ctx context.Context, namespace string, metricSelector labels.Selector, info provider.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
	klog.Info(fmt.Sprintf("Get metric by selector for external metric, Info %v namespace %s metricSelector %s", info, namespace, metricSelector.String()))

	if !IsLocalExternalMetric(info, p.client) {
		if p.remoteAdapter != nil {
			return p.remoteAdapter.GetExternalMetric(ctx, namespace, metricSelector, info)
		} else {
			return nil, apiErrors.NewServiceUnavailable("not supported")
		}
	}
	if strings.HasPrefix(info.Metric, "crane") {
		prediction, err := GetPrediction(ctx, p.client, namespace, metricSelector)
		if err != nil {
			return nil, err
		}

		isPredicting := false
		// check prediction is ongoing
		if prediction.Status.Conditions != nil {
			for _, condition := range prediction.Status.Conditions {
				if condition.Type == string(predictionapi.TimeSeriesPredictionConditionReady) && condition.Status == metav1.ConditionTrue {
					isPredicting = true
				}
			}
		}

		if !isPredicting {
			return nil, fmt.Errorf("TimeSeriesPrediction is not ready. ")
		}

		var timeSeries *predictionapi.MetricTimeSeries
		for _, metricStatus := range prediction.Status.PredictionMetrics {
			if metricStatus.ResourceIdentifier == info.Metric && len(metricStatus.Prediction) == 1 {
				timeSeries = metricStatus.Prediction[0]
			}
		}
		// check time series for current metric is empty
		if timeSeries == nil {
			return nil, fmt.Errorf("TimeSeries is empty, metric name %s", info.Metric)
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

		klog.Infof("Provide external metric %s average value %f.", info.Metric, largestMetricValue.value)

		return &external_metrics.ExternalMetricValueList{Items: []external_metrics.ExternalMetricValue{
			{
				MetricName: info.Metric,
				Timestamp:  metav1.Now(),
				Value:      *resource.NewQuantity(int64(largestMetricValue.value), resource.DecimalSI),
			},
		}}, nil
	}

	var ehpa autoscalingapi.EffectiveHorizontalPodAutoscaler

	err := p.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: info.Metric}, &ehpa)
	if err != nil {
		return &external_metrics.ExternalMetricValueList{}, err
	}
	// Find the cron metric scaler
	cronScalers := GetCronScalersForEHPA(&ehpa)
	var activeScalers []*CronScaler
	var errs []error
	for _, cronScaler := range cronScalers {
		isActive, err := cronScaler.IsActive(ctx, time.Now())
		if err != nil {
			errs = append(errs, err)
		}
		if isActive {
			activeScalers = append(activeScalers, cronScaler)
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("%v", errs)
	}
	replicas := DefaultCronTargetMetricValue
	if len(activeScalers) == 0 {
		// No active cron now, there are two cases:
		// 1. no other hpa metrics work with cron together, then return current workload replicas to keep the original desired replicas
		// 2. other hpa metrics work with cron together, then return min value to remove the cron impact for other metrics.
		// when cron is working with other metrics together, it should not return workload's original desired replicas,
		// because there maybe other metrics want to trigger the workload to scale in.
		// hpa controller select max replicas computed by all metrics(this is hpa default policy in hard code), cron will impact the hpa.
		// so we should remove the cron effect when cron is not active, it should return min value.
		scale, _, err := utils.GetScale(ctx, p.restMapper, p.scaler, namespace, ehpa.Spec.ScaleTargetRef)
		if err != nil {
			klog.Errorf("Failed to get scale: %v", err)
			return nil, err
		}
		// no other hpa metrics work with cron together, keep the workload desired replicas
		replicas = scale.Spec.Replicas

		if !utils.IsEHPAPredictionEnabled(&ehpa) {
			hpaList := &autoscalingv2.HorizontalPodAutoscalerList{}
			opts := []client.ListOption{
				client.MatchingLabels(map[string]string{known.EffectiveHorizontalPodAutoscalerUidLabel: string(ehpa.UID)}),
			}
			err := p.client.List(ctx, hpaList, opts...)
			if err != nil {
				return nil, err
			}
			// other hpa metrics work with cron together
			// excludes the cron metric itself
			if len(hpaList.Items) >= 0 && len(hpaList.Items[0].Spec.Metrics) > 1 {
				replicas = DefaultCronTargetMetricValue
			}
		} else {
			// other hpa metrics work with cron together
			replicas = DefaultCronTargetMetricValue
		}
	} else {
		// Has active ones. Basically, there should not be more then one active cron at the same time period, it is not a best practice.
		// we use the largest targetReplicas specified in cron spec.
		for _, activeScaler := range activeScalers {
			if activeScaler.TargetSize() >= replicas {
				replicas = activeScaler.TargetSize()
			}
		}
	}

	return &external_metrics.ExternalMetricValueList{Items: []external_metrics.ExternalMetricValue{
		{
			MetricName: info.Metric,
			Timestamp:  metav1.Now(),
			Value:      *resource.NewQuantity(int64(replicas), resource.DecimalSI),
		},
	}}, nil
}

// ListAllExternalMetrics return external cron metrics
// Fetch metrics from cache directly to avoid the performance issue for apiserver when the metrics is large, because this api is called frequently.
func (p *ExternalMetricProvider) ListAllExternalMetrics() []provider.ExternalMetricInfo {
	klog.Info("List all external metrics")

	metricInfos := ListAllLocalExternalMetrics(p.client)

	if p.remoteAdapter != nil {
		metricInfos = append(metricInfos, p.remoteAdapter.ListAllExternalMetrics()...)
	}
	return metricInfos
}

func ListAllLocalExternalMetrics(client client.Client) []provider.ExternalMetricInfo {
	var metricInfos []provider.ExternalMetricInfo
	var ehpaList autoscalingapi.EffectiveHorizontalPodAutoscalerList
	err := client.List(context.TODO(), &ehpaList)
	if err != nil {
		klog.Errorf("Failed to list ehpa: %v", err)
		return metricInfos
	}
	for _, ehpa := range ehpaList.Items {
		if CronEnabled(&ehpa) {
			metricInfos = append(metricInfos, provider.ExternalMetricInfo{Metric: ehpa.Name})
		}
	}

	return metricInfos
}

func IsLocalExternalMetric(metricInfo provider.ExternalMetricInfo, client client.Client) bool {
	for _, info := range ListAllLocalExternalMetrics(client) {
		if info.Metric == metricInfo.Metric {
			return true
		}
	}

	return false
}

func CronEnabled(ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler) bool {
	return len(ehpa.Spec.Crons) > 0
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
