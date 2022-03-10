package advisor

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/montanaflynn/stats"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/recommend/types"
	"github.com/gocrane/crane/pkg/utils"
)

var _ Advisor = &EHPAAdvisor{}

type EHPAAdvisor struct {
	*types.Context
}

func (a *EHPAAdvisor) Advise(proposed *types.ProposedRecommendation) error {
	p := a.Predictors[predictionapi.AlgorithmTypeDSP]

	resourceCpu := corev1.ResourceCPU
	namespace := a.Recommendation.Spec.TargetRef.Namespace
	if len(namespace) == 0 {
		namespace = DefaultNamespace
	}
	cpuQueryExpr := ResourceToPromQueryExpr(namespace, a.Recommendation.Spec.TargetRef.Name, &resourceCpu)

	klog.V(4).Infof("EHPAAdvisor CpuQuery %s Recommendation %s", cpuQueryExpr, klog.KObj(a.Recommendation))
	timeNow := time.Now()
	tsList, err := a.DataSource.QueryTimeSeries(cpuQueryExpr, timeNow.Add(-time.Hour*24*7), timeNow, time.Minute)
	if err != nil {
		return fmt.Errorf("EHPAAdvisor query historic metrics failed: %v ", err)
	}
	if len(tsList) != 1 {
		return fmt.Errorf("EHPAAdvisor query historic metrics data is unexpected, List length is %d ", len(tsList))
	}

	cpuConfig := getPredictionCpuConfig(cpuQueryExpr)
	tsListPrediction, err := utils.QueryPredictedTimeSeriesOnce(p, fmt.Sprintf(callerFormat, a.Recommendation.UID),
		getPredictionCpuConfig(cpuQueryExpr),
		cpuQueryExpr,
		timeNow,
		timeNow.Add(time.Hour*24*7))
	if err != nil {
		return fmt.Errorf("EHPAAdvisor query predicted time series failed: %v ", err)
	}

	if len(tsListPrediction) != 1 {
		return fmt.Errorf("EHPAAdvisor prediction metrics data is unexpected, List length is %d ", len(tsListPrediction))
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

	for _, sample := range tsListPrediction[0].Samples {
		cpuUsages = append(cpuUsages, sample.Value)
		if sample.Value > cpuMax {
			cpuMax = sample.Value
		}
	}

	err = a.checkMinCpuUsageThreshold(cpuMax)
	if err != nil {
		return fmt.Errorf("EHPAAdvisor checkMinCpuUsageThreshold failed: %v", err)
	}

	err = a.checkFluctuation(tsListPrediction)
	if err != nil {
		return fmt.Errorf("EHPAAdvisor checkFluctuation failed: %v", err)
	}

	minReplicas, err := a.proposeMinReplicas()
	if err != nil {
		return fmt.Errorf("EHPAAdvisor proposeMinReplicas failed: %v", err)
	}

	targetUtilization, err := a.proposeTargetUtilization()
	if err != nil {
		return fmt.Errorf("EHPAAdvisor proposeTargetUtilization failed: %v", err)
	}

	maxReplicas, err := a.proposeMaxReplicas(cpuUsages, targetUtilization, minReplicas)
	if err != nil {
		return fmt.Errorf("EHPAAdvisor proposeMaxReplicas failed: %v", err)
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
		Prediction: &autoscalingapi.Prediction{
			PredictionWindowSeconds: &defaultPredictionWindow,
			PredictionAlgorithm: &autoscalingapi.PredictionAlgorithm{
				AlgorithmType: predictionapi.AlgorithmTypeDSP,
				DSP:           cpuConfig.DSP,
			},
		},
	}

	referenceHpa, err := strconv.ParseBool(a.Context.ConfigProperties["ehpa.reference-hpa"])
	if err != nil {
		return err
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

	proposed.EffectiveHPA = proposedEHPA
	return nil
}

func (a *EHPAAdvisor) Name() string {
	return "EHPAAdvisor"
}

// checkMinCpuUsageThreshold check if the max cpu for target is reach to ehpa.min-cpu-usage-threshold
func (a *EHPAAdvisor) checkMinCpuUsageThreshold(cpuMax float64) error {
	minCpuUsageThreshold, err := strconv.ParseFloat(a.Context.ConfigProperties["ehpa.min-cpu-usage-threshold"], 64)
	if err != nil {
		return err
	}

	klog.V(4).Infof("EHPAAdvisor checkMinCpuUsageThreshold, cpuMax %f threshold %f", cpuMax, minCpuUsageThreshold)
	if cpuMax < minCpuUsageThreshold {
		return fmt.Errorf("Target cpuusage %f is under ehpa.min-cpu-usage-threshold %f. ", cpuMax, minCpuUsageThreshold)
	}

	return nil
}

// checkFluctuation check if the time series fluctuation is reach to ehpa.fluctuation-threshold
func (a *EHPAAdvisor) checkFluctuation(predictionTs []*common.TimeSeries) error {
	fluctuationThreshold, err := strconv.ParseFloat(a.Context.ConfigProperties["ehpa.fluctuation-threshold"], 64)
	if err != nil {
		return err
	}

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
			return err
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

	if medianMin == 0 {
		return fmt.Errorf("Mean cpu usage is zero. ")
	}

	klog.V(4).Infof("EHPAAdvisor checkFluctuation, medianMax %f medianMin %f medianUsages %v", medianMax, medianMin, medianUsages)
	fluctuation := medianMax / medianMin
	if fluctuation < fluctuationThreshold {
		return fmt.Errorf("Target cpu fluctuation %f is under ehpa.fluctuation-threshold %f. ", fluctuation, fluctuationThreshold)
	}

	return nil
}

// proposeTargetUtilization use the 99 percentile cpu usage to propose target utilization,
// since we think if pod have reach the top usage before, maybe this is a suitable target to running.
// Considering too high or too low utilization are both invalid, we will be capping target utilization finally.
func (a *EHPAAdvisor) proposeTargetUtilization() (int32, error) {
	minCpuTargetUtilization, err := strconv.ParseInt(a.Context.ConfigProperties["ehpa.min-cpu-target-utilization"], 10, 32)
	if err != nil {
		return 0, err
	}

	maxCpuTargetUtilization, err := strconv.ParseInt(a.Context.ConfigProperties["ehpa.max-cpu-target-utilization"], 10, 32)
	if err != nil {
		return 0, err
	}

	percentilePredictor := a.Predictors[predictionapi.AlgorithmTypePercentile]

	var cpuUsage float64
	// use percentile algo to get the 99 percentile cpu usage for this target
	for _, container := range a.PodTemplate.Spec.Containers {
		queryExpr := fmt.Sprintf(cpuQueryExprTemplate, container.Name, a.Recommendation.Spec.TargetRef.Namespace, a.Recommendation.Spec.TargetRef.Name)
		cpuConfig := makeCpuConfig(a.ConfigProperties)
		tsList, err := utils.QueryPredictedValuesOnce(a.Recommendation,
			percentilePredictor,
			fmt.Sprintf(callerFormat, a.Recommendation.UID),
			cpuConfig,
			queryExpr)
		if err != nil {
			return 0, err
		}
		if len(tsList) < 1 || len(tsList[0].Samples) < 1 {
			return 0, fmt.Errorf("no value retured for queryExpr: %s", queryExpr)
		}
		cpuUsage += tsList[0].Samples[0].Value
	}

	requestsPod, err := utils.CalculatePodTemplateRequests(a.PodTemplate, corev1.ResourceCPU)
	if err != nil {
		return 0, err
	}

	klog.V(4).Infof("EHPAAdvisor propose targetUtilization, cpuUsage %f requestsPod %d", cpuUsage, requestsPod)
	targetUtilization := int32(math.Ceil((cpuUsage * 1000 / float64(requestsPod)) * 100))

	// capping
	if targetUtilization < int32(minCpuTargetUtilization) {
		targetUtilization = int32(minCpuTargetUtilization)
	}

	// capping
	if targetUtilization > int32(maxCpuTargetUtilization) {
		targetUtilization = int32(maxCpuTargetUtilization)
	}

	return targetUtilization, nil
}

// proposeMinReplicas calculate min replicas based on ehpa.default-min-replicas
func (a *EHPAAdvisor) proposeMinReplicas() (int32, error) {
	defaultMinReplicas, err := strconv.ParseInt(a.Context.ConfigProperties["ehpa.default-min-replicas"], 10, 32)
	if err != nil {
		return 0, err
	}

	minReplicas := int32(defaultMinReplicas)

	// minReplicas should be larger than 0
	if minReplicas < 1 {
		minReplicas = 1
	}

	return minReplicas, nil
}

// proposeMaxReplicas use max cpu usage to compare with target pod cpu usage to get the max replicas.
func (a *EHPAAdvisor) proposeMaxReplicas(cpuUsages []float64, targetUtilization int32, minReplicas int32) (int32, error) {
	maxReplicasFactor, err := strconv.ParseFloat(a.Context.ConfigProperties["ehpa.max-replicas-factor"], 64)
	if err != nil {
		return 0, err
	}
	// use percentile to deburring data
	p95thCpu, err := stats.Percentile(cpuUsages, 95)
	if err != nil {
		return 0, err
	}
	requestsPod, err := utils.CalculatePodTemplateRequests(a.PodTemplate, corev1.ResourceCPU)
	if err != nil {
		return 0, err
	}

	klog.V(4).Infof("EHPAAdvisor proposeMaxReplicas, p95thCpu %f requestsPod %d targetUtilization %d", p95thCpu, requestsPod, targetUtilization)

	// request * targetUtilization is the target average cpu usage, use total p95thCpu to divide, we can get the expect max replicas.
	calcMaxReplicas := (p95thCpu * 100 * 1000 * maxReplicasFactor) / float64(int32(requestsPod)*targetUtilization)
	maxReplicas := int32(math.Ceil(calcMaxReplicas))

	// maxReplicas should be always larger than minReplicas
	if maxReplicas < minReplicas {
		maxReplicas = minReplicas
	}

	return maxReplicas, nil
}

func getPredictionCpuConfig(expr string) *config.Config {
	return &config.Config{
		Expression: &predictionapi.ExpressionQuery{Expression: expr},
		DSP:        &predictionapi.DSP{}, // use default DSP config will let prediction tuning by itself
	}
}

func ResourceToPromQueryExpr(namespace string, name string, resourceName *corev1.ResourceName) string {
	switch *resourceName {
	case corev1.ResourceCPU:
		return fmt.Sprintf(config.WorkloadCpuUsagePromQLFmtStr, namespace, name, "5m")
	case corev1.ResourceMemory:
		return fmt.Sprintf(config.WorkloadMemUsagePromQLFmtStr, namespace, name)
	}

	return ""
}
