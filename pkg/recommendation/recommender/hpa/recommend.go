package hpa

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/montanaflynn/stats"
	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/recommend/types"
	"github.com/gocrane/crane/pkg/recommendation/framework"
	"github.com/gocrane/crane/pkg/utils"
)

const callerFormat = "HPARecommendationCaller-%s-%s"

func (rr *HPARecommender) PreRecommend(ctx *framework.RecommendationContext) error {
	return rr.ReplicasRecommender.PreRecommend(ctx)
}

func (rr *HPARecommender) Recommend(ctx *framework.RecommendationContext) error {
	return rr.ReplicasRecommender.Recommend(ctx)
}

// Policy add some logic for result of recommend phase.
func (rr *HPARecommender) Policy(ctx *framework.RecommendationContext) error {
	predictable := true

	if len(ctx.ResultValues) != 1 {
		klog.Warningf("%s: prediction metrics data is unexpected, List length is %d ", ctx.String(), len(ctx.ResultValues))
		predictable = false
	}

	if rr.PredictableEnabled && !predictable {
		return fmt.Errorf("cannot predict target")
	}

	minReplicas, cpuMax, percentileCpu, err := rr.GetMinReplicas(ctx)
	if err != nil {
		return err
	}

	err = rr.checkMinCpuUsageThreshold(cpuMax)
	if err != nil {
		return fmt.Errorf("checkMinCpuUsageThreshold failed: %v", err)
	}

	medianMin, medianMax, err := rr.minMaxMedians(ctx.InputValues)
	if err != nil {
		return fmt.Errorf("minMaxMedians failed: %v", err)
	}

	err = rr.checkFluctuation(medianMin, medianMax)
	if err != nil {
		return fmt.Errorf("%s checkFluctuation failed: %v", rr.Name(), err)
	}

	targetUtilization, _, err := rr.proposeTargetUtilization(ctx)
	if err != nil {
		return fmt.Errorf("proposeTargetUtilization failed: %v", err)
	}

	maxReplicas, err := rr.proposeMaxReplicas(&ctx.PodTemplate, percentileCpu, targetUtilization, minReplicas)
	if err != nil {
		return fmt.Errorf("proposeMaxReplicas failed: %v", err)
	}

	defaultPredictionWindow := int32(3600)
	resourceCpu := corev1.ResourceCPU

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
				DSP:           ctx.AlgorithmConfig.DSP,
			},
		}
	}

	// get metric spec from existing hpa and use them
	if rr.ReferenceHpaEnabled && ctx.HPA != nil {
		for _, metricSpec := range ctx.HPA.Spec.Metrics {
			// don't use resource cpu, since we already configuration it before
			if metricSpec.Type == autoscalingv2.ResourceMetricSourceType && metricSpec.Resource != nil && metricSpec.Resource.Name == resourceCpu {
				continue
			}

			proposedEHPA.Metrics = append(proposedEHPA.Metrics, metricSpec)
		}
	}

	result := types.ProposedRecommendation{
		EffectiveHPA: proposedEHPA,
	}

	resultBytes, err := yaml.Marshal(result)
	if err != nil {
		return fmt.Errorf("%s marshal result failed: %v", rr.Name(), err)
	}

	ctx.Recommendation.Status.RecommendedValue = string(resultBytes)
	if ctx.EHPA == nil {
		ctx.Recommendation.Status.Action = "Create"

		newEhpa := &autoscalingapi.EffectiveHorizontalPodAutoscaler{
			TypeMeta: metav1.TypeMeta{
				Kind:       "EffectiveHorizontalPodAutoscaler",
				APIVersion: autoscalingapi.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ctx.Recommendation.Spec.TargetRef.Namespace,
				Name:      ctx.Recommendation.Spec.TargetRef.Name,
			},
			Spec: autoscalingapi.EffectiveHorizontalPodAutoscalerSpec{
				MinReplicas:   proposedEHPA.MinReplicas,
				MaxReplicas:   *proposedEHPA.MaxReplicas,
				Metrics:       proposedEHPA.Metrics,
				ScaleStrategy: autoscalingapi.ScaleStrategyPreview,
				Prediction:    proposedEHPA.Prediction,
				ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
					Kind:       ctx.Recommendation.Spec.TargetRef.Kind,
					APIVersion: ctx.Recommendation.Spec.TargetRef.APIVersion,
					Name:       ctx.Recommendation.Spec.TargetRef.Name,
				},
			},
		}

		newEhpaBytes, err := json.Marshal(newEhpa)
		if err != nil {
			return fmt.Errorf("marshal ehpa failed %s. ", err)
		}
		ctx.Recommendation.Status.RecommendedInfo = string(newEhpaBytes)
	} else {
		ctx.Recommendation.Status.Action = "Patch"

		patchEhpa := &autoscalingapi.EffectiveHorizontalPodAutoscaler{
			Spec: autoscalingapi.EffectiveHorizontalPodAutoscalerSpec{
				MinReplicas: proposedEHPA.MinReplicas,
				MaxReplicas: *proposedEHPA.MaxReplicas,
				Metrics:     proposedEHPA.Metrics,
			},
		}

		patchEhpaBytes, err := json.Marshal(patchEhpa)
		if err != nil {
			return fmt.Errorf("marshal ehpa failed %s. ", err)
		}
		ctx.Recommendation.Status.RecommendedInfo = string(patchEhpaBytes)
		ctx.Recommendation.Status.TargetRef = corev1.ObjectReference{
			Namespace:  ctx.Recommendation.Spec.TargetRef.Namespace,
			Name:       ctx.Recommendation.Spec.TargetRef.Name,
			Kind:       "EffectiveHorizontalPodAutoscaler",
			APIVersion: autoscalingapi.GroupVersion.String(),
		}
	}

	return nil
}

// checkMinCpuUsageThreshold check if the max cpu for target is reach to replicas.min-cpu-usage-threshold
func (rr *HPARecommender) checkMinCpuUsageThreshold(cpuMax float64) error {
	klog.V(4).Infof("%s checkMinCpuUsageThreshold, cpuMax %f threshold %f", rr.Name(), cpuMax, rr.MinCpuUsageThreshold)
	if cpuMax < rr.MinCpuUsageThreshold {
		return fmt.Errorf("target cpuusage %f is under replicas.min-cpu-usage-threshold %f. ", cpuMax, rr.MinCpuUsageThreshold)
	}

	return nil
}

func (rr *HPARecommender) minMaxMedians(predictionTs []*common.TimeSeries) (float64, float64, error) {
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

	klog.V(4).Infof("%s minMaxMedians medianMax %f, medianMin %f, medianUsages %v", rr.Name(), medianMax, medianMin, medianUsages)

	return medianMin, medianMax, nil
}

// checkFluctuation check if the time series fluctuation is reach to replicas.fluctuation-threshold
func (rr *HPARecommender) checkFluctuation(medianMin, medianMax float64) error {
	fluctuationThreshold, err := strconv.ParseFloat(rr.Config["fluctuation-threshold"], 64)
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
func (rr *HPARecommender) proposeTargetUtilization(ctx *framework.RecommendationContext) (int32, int64, error) {
	percentilePredictor := ctx.PredictorMgr.GetPredictor(predictionapi.AlgorithmTypePercentile)

	var cpuUsage float64
	// use percentile algo to get the 99 percentile cpu usage for this target
	for _, container := range ctx.PodTemplate.Spec.Containers {
		caller := fmt.Sprintf(callerFormat, klog.KObj(ctx.Recommendation), ctx.Recommendation.UID)
		metricNamer := metricnaming.ResourceToContainerMetricNamer(ctx.Recommendation.Spec.TargetRef.Namespace, ctx.Recommendation.Spec.TargetRef.APIVersion,
			ctx.Recommendation.Spec.TargetRef.Kind, ctx.Recommendation.Spec.TargetRef.Name, container.Name, corev1.ResourceCPU, caller)
		cpuConfig := &config.Config{
			Percentile: &predictionapi.Percentile{
				Aggregated:        true,
				HistoryLength:     "168h",
				SampleInterval:    "1m",
				MarginFraction:    "0.15",
				TargetUtilization: "1.0",
				Percentile:        "0.99",
				Histogram: predictionapi.HistogramConfig{
					HalfLife:   "24h",
					BucketSize: "0.1",
					MaxValue:   "100",
				},
			},
		}
		tsList, err := utils.QueryPredictedValuesOnce(ctx.Recommendation,
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

	requestTotal, err := utils.CalculatePodTemplateRequests(&ctx.PodTemplate, corev1.ResourceCPU)
	if err != nil {
		return 0, 0, err
	}

	klog.V(4).Infof("propose targetUtilization, cpuUsage %f requestsPod %d", cpuUsage, requestTotal)
	targetUtilization := int32(math.Ceil((cpuUsage * 1000 / float64(requestTotal)) * 100))

	// capping
	if targetUtilization < int32(rr.MinCpuTargetUtilization) {
		targetUtilization = int32(rr.MinCpuTargetUtilization)
	}

	// capping
	if targetUtilization > int32(rr.MaxCpuTargetUtilization) {
		targetUtilization = int32(rr.MaxCpuTargetUtilization)
	}

	return targetUtilization, requestTotal, nil
}

// proposeMaxReplicas use max cpu usage to compare with target pod cpu usage to get the max replicas.
func (rr *HPARecommender) proposeMaxReplicas(podTemplate *corev1.PodTemplateSpec, percentileCpu float64, targetUtilization int32, minReplicas int32) (int32, error) {
	requestsPod, err := utils.CalculatePodTemplateRequests(podTemplate, corev1.ResourceCPU)
	if err != nil {
		return 0, err
	}

	klog.V(4).Infof("proposeMaxReplicas, percentileCpu %f requestsPod %d targetUtilization %d", percentileCpu, requestsPod, targetUtilization)

	// request * targetUtilization is the target average cpu usage, use total p95thCpu to divide, we can get the expect max replicas.
	calcMaxReplicas := (percentileCpu * 100 * 1000 * rr.MaxReplicasFactor) / float64(int32(requestsPod)*targetUtilization)
	maxReplicas := int32(math.Ceil(calcMaxReplicas))

	// maxReplicas should be always larger than minReplicas
	if maxReplicas < minReplicas {
		maxReplicas = minReplicas
	}

	return maxReplicas, nil
}
