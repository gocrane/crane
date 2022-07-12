package advisor

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/montanaflynn/stats"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/metricquery"
	"github.com/gocrane/crane/pkg/metrics"
	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/recommend/types"
	"github.com/gocrane/crane/pkg/utils"
)

var _ Advisor = &ReplicasAdvisor{}

type ReplicasAdvisor struct {
	*types.Context
}

func (a *ReplicasAdvisor) Advise(proposed *types.ProposedRecommendation) error {
	p := a.PredictorMgr.GetPredictor(predictionapi.AlgorithmTypeDSP)
	if p == nil {
		return fmt.Errorf("predictor %v not found", predictionapi.AlgorithmTypeDSP)
	}

	predictableEnabled, err := strconv.ParseBool(a.Context.ConfigProperties["replicas.predictable"])
	if err != nil {
		predictableEnabled = false
	}

	resourceCpu := corev1.ResourceCPU
	target := a.Recommendation.Spec.TargetRef.DeepCopy()
	if len(target.Namespace) == 0 {
		target.Namespace = DefaultNamespace
	}

	labelSelector, err := GetTargetLabelSelector(target, a.Scale, a.DaemonSet)
	if err != nil {
		return err
	}
	caller := fmt.Sprintf(callerFormat, klog.KObj(a.Recommendation), a.Recommendation.UID)
	metricNamer := ResourceToWorkloadMetricNamer(target, &resourceCpu, labelSelector, caller)
	if err := metricNamer.Validate(); err != nil {
		return err
	}
	klog.V(4).Infof("ReplicasAdvisor CpuQuery %s Recommendation %s", metricNamer.BuildUniqueKey(), klog.KObj(a.Recommendation))
	timeNow := time.Now()
	tsList, err := a.DataSource.QueryTimeSeries(metricNamer, timeNow.Add(-time.Hour*24*7), timeNow, time.Minute)
	if err != nil {
		return fmt.Errorf("ReplicasAdvisor query historic metrics failed: %v ", err)
	}
	if len(tsList) != 1 {
		return fmt.Errorf("ReplicasAdvisor query historic metrics data is unexpected, List length is %d ", len(tsList))
	}

	predictable := true
	cpuConfig := getPredictionCpuConfig()
	tsListPrediction, err := utils.QueryPredictedTimeSeriesOnce(p, caller,
		getPredictionCpuConfig(),
		metricNamer,
		timeNow,
		timeNow.Add(time.Hour*24*7))
	if err != nil {
		klog.Warningf("ReplicasAdvisor query predicted time series failed: %v ", err)
		predictable = false
	}

	if len(tsListPrediction) != 1 {
		klog.Warningf("ReplicasAdvisor prediction metrics data is unexpected, List length is %d ", len(tsListPrediction))
		predictable = false
	}

	if predictableEnabled && !predictable {
		return fmt.Errorf("ReplicasAdvisor cannot predict target: %v ", err)
	}

	var cpuMax float64
	var cpuUsages []float64
	// combine values from historic and prediction
	for _, sample := range tsList[0].Samples {
		cpuUsages = append(cpuUsages, sample.Value)
		if sample.Value > cpuMax {
			cpuMax = sample.Value
		}
	}

	if predictable {
		for _, sample := range tsListPrediction[0].Samples {
			cpuUsages = append(cpuUsages, sample.Value)
			if sample.Value > cpuMax {
				cpuMax = sample.Value
			}
		}
	}

	cpuPercentile, err := strconv.ParseFloat(a.Context.ConfigProperties["replicas.cpu-percentile"], 64)
	if err != nil {
		return fmt.Errorf("ReplicasAdvisor parse replicas.cpu-percentile failed: %v", err)
	}

	// use percentile to deburring data
	percentileCpu, err := stats.Percentile(cpuUsages, cpuPercentile)
	if err != nil {
		return fmt.Errorf("ReplicasAdvisor get percentileCpu failed: %v", err)
	}

	err = a.checkMinCpuUsageThreshold(cpuMax)
	if err != nil {
		return fmt.Errorf("ReplicasAdvisor checkMinCpuUsageThreshold failed: %v", err)
	}

	medianMin, medianMax, err := a.minMaxMedians(tsList)
	if err != nil {
		return fmt.Errorf("ReplicasAdvisor minMaxMedians failed: %v", err)
	}

	err = a.checkFluctuation(medianMin, medianMax)
	if err != nil {
		return fmt.Errorf("ReplicasAdvisor checkFluctuation failed: %v", err)
	}

	targetUtilization, requestTotal, err := a.proposeTargetUtilization()
	if err != nil {
		return fmt.Errorf("ReplicasAdvisor proposeTargetUtilization failed: %v", err)
	}

	minReplicas, err := a.proposeMinReplicas(percentileCpu, requestTotal)
	if err != nil {
		return fmt.Errorf("ReplicasAdvisor proposeMinReplicas failed: %v", err)
	}

	maxReplicas, err := a.proposeMaxReplicas(percentileCpu, targetUtilization, minReplicas)
	if err != nil {
		return fmt.Errorf("ReplicasAdvisor proposeMaxReplicas failed: %v", err)
	}

	defaultPredictionWindow := int32(3600)
	proposedEHPA := &types.EffectiveHorizontalPodAutoscalerRecommendation{
		MaxReplicas: &maxReplicas,
		MinReplicas: &minReplicas,
		Metrics: []autoscalingv2.MetricSpec{
			{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: resourceCpu,
					Target: autoscalingv2.MetricTarget{
						Type:               autoscalingv2.UtilizationMetricType,
						AverageUtilization: &targetUtilization,
					},
				},
			},
		},
	}

	if predictable {
		proposedEHPA.Prediction = &autoscalingapi.Prediction{
			PredictionWindowSeconds: &defaultPredictionWindow,
			PredictionAlgorithm: &autoscalingapi.PredictionAlgorithm{
				AlgorithmType: predictionapi.AlgorithmTypeDSP,
				DSP:           cpuConfig.DSP,
			},
		}
	}

	referenceHpa, err := strconv.ParseBool(a.Context.ConfigProperties["replicas.reference-hpa"])
	if err != nil {
		referenceHpa = false
	}

	// get metric spec from existing hpa and use them
	if referenceHpa && a.HPA != nil {
		for _, metricSpec := range a.HPA.Spec.Metrics {
			// don't use resource cpu, since we already configuration it before
			if metricSpec.Type == autoscalingv2.ResourceMetricSourceType && metricSpec.Resource != nil && metricSpec.Resource.Name == resourceCpu {
				continue
			}

			proposedEHPA.Metrics = append(proposedEHPA.Metrics, metricSpec)
		}
	}

	replicasRecommendation := &types.ReplicasRecommendation{
		Replicas: &minReplicas,
	}

	a.recordReplicasRecommendation(minReplicas)

	proposed.EffectiveHPA = proposedEHPA
	proposed.ReplicasRecommendation = replicasRecommendation
	return nil
}

func (a *ReplicasAdvisor) Name() string {
	return "ReplicasAdvisor"
}

// checkMinCpuUsageThreshold check if the max cpu for target is reach to replicas.min-cpu-usage-threshold
func (a *ReplicasAdvisor) checkMinCpuUsageThreshold(cpuMax float64) error {
	minCpuUsageThreshold, err := strconv.ParseFloat(a.Context.ConfigProperties["replicas.min-cpu-usage-threshold"], 64)
	if err != nil {
		return err
	}

	klog.V(4).Infof("ReplicasAdvisor checkMinCpuUsageThreshold, cpuMax %f threshold %f", cpuMax, minCpuUsageThreshold)
	if cpuMax < minCpuUsageThreshold {
		return fmt.Errorf("target cpuusage %f is under replicas.min-cpu-usage-threshold %f. ", cpuMax, minCpuUsageThreshold)
	}

	return nil
}

func (a *ReplicasAdvisor) minMaxMedians(predictionTs []*common.TimeSeries) (float64, float64, error) {
	// aggregate with time's hour
	cpuUsagePredictionMap := make(map[int][]float64)
	for _, sample := range predictionTs[0].Samples {
		sampleTime := time.Unix(sample.Timestamp, 0)
		if _, exist := cpuUsagePredictionMap[sampleTime.Hour()]; exist {
			cpuUsagePredictionMap[sampleTime.Hour()] = append(cpuUsagePredictionMap[sampleTime.Hour()], sample.Value)
		} else {
			newUsageInHour := make([]float64, 0)
			newUsageInHour = append(newUsageInHour, sample.Value)
			cpuUsagePredictionMap[sampleTime.Hour()] = newUsageInHour
		}
	}

	// use median to deburring data
	var medianUsages []float64
	for _, usageInHour := range cpuUsagePredictionMap {
		medianUsage, err := stats.Median(usageInHour)
		if err != nil {
			return 0., 0., err
		}
		medianUsages = append(medianUsages, medianUsage)
	}

	medianMax := math.SmallestNonzeroFloat64
	medianMin := math.MaxFloat64
	for _, value := range medianUsages {
		if value > medianMax {
			medianMax = value
		}

		if value < medianMin {
			medianMin = value
		}
	}

	klog.V(4).Infof("ReplicasAdvisor minMaxMedians medianMax %f, medianMin %f, medianUsages %v", medianMax, medianMin, medianUsages)

	return medianMin, medianMax, nil
}

// checkFluctuation check if the time series fluctuation is reach to replicas.fluctuation-threshold
func (a *ReplicasAdvisor) checkFluctuation(medianMin, medianMax float64) error {
	fluctuationThreshold, err := strconv.ParseFloat(a.Context.ConfigProperties["replicas.fluctuation-threshold"], 64)
	if err != nil {
		return err
	}

	if medianMin == 0 {
		medianMin = 0.1 // use a small value to continue calculate
	}

	fluctuation := medianMax / medianMin
	if fluctuation < fluctuationThreshold {
		return fmt.Errorf("target cpu fluctuation %f is under replicas.fluctuation-threshold %f. ", fluctuation, fluctuationThreshold)
	}

	return nil
}

// proposeTargetUtilization use the 99 percentile cpu usage to propose target utilization,
// since we think if pod have reach the top usage before, maybe this is a suitable target to running.
// Considering too high or too low utilization are both invalid, we will be capping target utilization finally.
func (a *ReplicasAdvisor) proposeTargetUtilization() (int32, int64, error) {
	minCpuTargetUtilization, err := strconv.ParseInt(a.Context.ConfigProperties["replicas.min-cpu-target-utilization"], 10, 32)
	if err != nil {
		return 0, 0, err
	}

	maxCpuTargetUtilization, err := strconv.ParseInt(a.Context.ConfigProperties["replicas.max-cpu-target-utilization"], 10, 32)
	if err != nil {
		return 0, 0, err
	}

	percentilePredictor := a.PredictorMgr.GetPredictor(predictionapi.AlgorithmTypePercentile)

	var cpuUsage float64
	// use percentile algo to get the 99 percentile cpu usage for this target
	for _, container := range a.PodTemplate.Spec.Containers {
		caller := fmt.Sprintf(callerFormat, klog.KObj(a.Recommendation), a.Recommendation.UID)
		metricNamer := ResourceToContainerMetricNamer(a.Recommendation.Spec.TargetRef.Namespace, a.Recommendation.Spec.TargetRef.APIVersion,
			a.Recommendation.Spec.TargetRef.Kind, a.Recommendation.Spec.TargetRef.Name, container.Name, corev1.ResourceCPU, caller)
		cpuConfig := makeCpuConfig(a.ConfigProperties)
		tsList, err := utils.QueryPredictedValuesOnce(a.Recommendation,
			percentilePredictor,
			caller,
			cpuConfig,
			metricNamer)
		if err != nil {
			return 0, 0, err
		}
		if len(tsList) < 1 || len(tsList[0].Samples) < 1 {
			return 0, 0, fmt.Errorf("no value retured for queryExpr: %s", metricNamer.BuildUniqueKey())
		}
		cpuUsage += tsList[0].Samples[0].Value
	}

	requestTotal, err := utils.CalculatePodTemplateRequests(a.PodTemplate, corev1.ResourceCPU)
	if err != nil {
		return 0, 0, err
	}

	klog.V(4).Infof("ReplicasAdvisor propose targetUtilization, cpuUsage %f requestsPod %d", cpuUsage, requestTotal)
	targetUtilization := int32(math.Ceil((cpuUsage * 1000 / float64(requestTotal)) * 100))

	// capping
	if targetUtilization < int32(minCpuTargetUtilization) {
		targetUtilization = int32(minCpuTargetUtilization)
	}

	// capping
	if targetUtilization > int32(maxCpuTargetUtilization) {
		targetUtilization = int32(maxCpuTargetUtilization)
	}

	return targetUtilization, requestTotal, nil
}

// proposeMinReplicas calculate min replicas based on replicas.default-min-replicas
func (a *ReplicasAdvisor) proposeMinReplicas(workloadCpu float64, requestTotal int64) (int32, error) {
	defaultMinReplicas, err := strconv.ParseInt(a.Context.ConfigProperties["replicas.default-min-replicas"], 10, 32)
	if err != nil {
		return 0, err
	}

	targetUtilization, err := strconv.ParseInt(a.Context.ConfigProperties["replicas.cpu-target-utilization"], 10, 32)
	if err != nil {
		return 0, err
	}

	minReplicas := int32(defaultMinReplicas)

	// minReplicas should be larger than 0
	if minReplicas < 1 {
		minReplicas = 1
	}

	min := int32(math.Ceil(workloadCpu / (float64(targetUtilization) / 100. * float64(requestTotal) / 1000.)))
	if min > minReplicas {
		minReplicas = min
	}

	return minReplicas, nil
}

// proposeMaxReplicas use max cpu usage to compare with target pod cpu usage to get the max replicas.
func (a *ReplicasAdvisor) proposeMaxReplicas(percentileCpu float64, targetUtilization int32, minReplicas int32) (int32, error) {
	maxReplicasFactor, err := strconv.ParseFloat(a.Context.ConfigProperties["replicas.max-replicas-factor"], 64)
	if err != nil {
		return 0, err
	}

	requestsPod, err := utils.CalculatePodTemplateRequests(a.PodTemplate, corev1.ResourceCPU)
	if err != nil {
		return 0, err
	}

	klog.V(4).Infof("ReplicasAdvisor proposeMaxReplicas, percentileCpu %f requestsPod %d targetUtilization %d", percentileCpu, requestsPod, targetUtilization)

	// request * targetUtilization is the target average cpu usage, use total p95thCpu to divide, we can get the expect max replicas.
	calcMaxReplicas := (percentileCpu * 100 * 1000 * maxReplicasFactor) / float64(int32(requestsPod)*targetUtilization)
	maxReplicas := int32(math.Ceil(calcMaxReplicas))

	// maxReplicas should be always larger than minReplicas
	if maxReplicas < minReplicas {
		maxReplicas = minReplicas
	}

	return maxReplicas, nil
}

func (a *ReplicasAdvisor) recordReplicasRecommendation(replicas int32) {
	labels := map[string]string{
		"apiversion": a.Recommendation.Spec.TargetRef.APIVersion,
		"owner_kind": a.Recommendation.Spec.TargetRef.Kind,
		"namespace":  a.Recommendation.Spec.TargetRef.Namespace,
		"owner_name": a.Recommendation.Spec.TargetRef.Name,
	}
	metrics.ReplicasRecommendation.With(labels).Set(float64(replicas))
}

func getPredictionCpuConfig() *config.Config {
	return &config.Config{
		DSP: &predictionapi.DSP{
			SampleInterval: "1m",
			HistoryLength:  "7d",
			Estimators:     predictionapi.Estimators{},
		},
	}
}

func GetTargetLabelSelector(target *corev1.ObjectReference, scale *v1.Scale, ds *appsv1.DaemonSet) (labels.Selector, error) {
	if target.Kind != "DaemonSet" {
		labelsMap, err := labels.ConvertSelectorToLabelsMap(scale.Status.Selector)
		if err != nil {
			return nil, err
		}
		return labelsMap.AsSelector(), nil
	} else {
		if ds != nil {
			labelsMap := labels.SelectorFromSet(ds.Spec.Selector.MatchLabels)
			return labelsMap, nil
		}
		return nil, fmt.Errorf("no daemonset specified")
	}
}

func ResourceToWorkloadMetricNamer(target *corev1.ObjectReference, resourceName *corev1.ResourceName, workloadLabelSelector labels.Selector, caller string) metricnaming.MetricNamer {
	// workload
	return &metricnaming.GeneralMetricNamer{
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
