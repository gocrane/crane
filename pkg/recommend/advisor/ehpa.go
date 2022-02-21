package advisor

import (
	"fmt"
	"math"
	"time"

	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/recommend/types"
	"github.com/gocrane/crane/pkg/utils"
)

var _ Advisor = &EHPAAdvisor{}

const ehpaCaller = "EffectiveHPACaller"

type EHPAAdvisor struct {
	*types.Context
}

func (a *EHPAAdvisor) Advise(proposed *types.ProposedRecommendation) error {
	p := a.Predictors[predictionapi.AlgorithmTypeDSP]

	mc := &config.MetricContext{}

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

	err = p.WithQuery(cpuQueryExpr, ehpaCaller)
	if err != nil {
		return err
	}
	cpuConfig := getPredictionCpuConfig(cpuQueryExpr)
	mc.WithConfig(cpuConfig)

	tsListPrediction, err := p.QueryPredictedTimeSeries(cpuQueryExpr, timeNow, timeNow.Add(time.Hour*24*7))
	if err != nil {
		return fmt.Errorf("EHPAAdvisor query predicted time series failed: %v ", err)
	}

	if len(tsListPrediction) != 1 {
		return fmt.Errorf("EHPAAdvisor prediction metrics data is unexpected, List length is %d ", len(tsListPrediction))
	}

	requestTotal, err := utils.CalculatePodRequests(a.Pods, resourceCpu)
	if err != nil {
		return fmt.Errorf("EHPAAdvisor calculate pod requests meet error %v ", err)
	}

	maxCpuUsage := math.SmallestNonzeroFloat64
	minCpuUsage := math.MaxFloat64
	// compute historic metric
	for _, sample := range tsList[0].Samples {
		if sample.Value > maxCpuUsage {
			maxCpuUsage = sample.Value
		}
		if sample.Value < minCpuUsage {
			minCpuUsage = sample.Value
		}
	}

	// compute prediction metric
	for _, sample := range tsListPrediction[0].Samples {
		if sample.Value > maxCpuUsage {
			maxCpuUsage = sample.Value
		}

		if sample.Value < minCpuUsage {
			minCpuUsage = sample.Value
		}
	}

	klog.V(4).Info("EHPAAdvisor maxCpuUsage %f minCpuUsage %f", maxCpuUsage, minCpuUsage)

	targetCpuUtilization := int32(50) // todo: configurable
	maxReplicasFactor := 1.2          // todo: configurable

	maxCpuUtilization := int32(int64(maxCpuUsage) * 1000 * 100 / requestTotal)
	proposedMaxRatio := float64(maxCpuUtilization) / float64(targetCpuUtilization)
	maxReplicasProposed := int32(math.Ceil(proposedMaxRatio * float64(a.ReadyPodNumber) * maxReplicasFactor))

	minCpuUtilization := int32(int64(minCpuUsage) * 1000 * 100 / requestTotal)
	proposedMinRatio := float64(minCpuUtilization) / float64(targetCpuUtilization)
	minReplicasProposed := int32(math.Ceil(proposedMinRatio * float64(a.ReadyPodNumber)))

	// minReplicasProposed should be larger than 0
	if minReplicasProposed < 1 {
		minReplicasProposed = 1
	}

	// maxReplicas should be always larger than minReplicas
	if maxReplicasProposed < minReplicasProposed {
		maxReplicasProposed = minReplicasProposed
	}

	defaultPredictionWindow := int32(3600)
	proposedEHPA := &types.EffectiveHorizontalPodAutoscalerRecommendation{
		MaxReplicas: &maxReplicasProposed,
		MinReplicas: &minReplicasProposed,
		Metrics: []autoscalingv2.MetricSpec{
			{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: resourceCpu,
					Target: autoscalingv2.MetricTarget{
						Type:               autoscalingv2.UtilizationMetricType,
						AverageUtilization: &targetCpuUtilization,
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

	proposed.EffectiveHPA = proposedEHPA
	return nil
}

func (a *EHPAAdvisor) Name() string {
	return "EHPAAdvisor"
}

func getPredictionCpuConfig(expr string) *config.Config {
	return &config.Config{
		Expression: &predictionapi.ExpressionQuery{Expression: expr},
		DSP: &predictionapi.DSP{
			SampleInterval: "1m",
			HistoryLength:  "5d",
			Estimators: predictionapi.Estimators{
				FFTEstimators: []*predictionapi.FFTEstimator{
					{MarginFraction: "0.05", LowAmplitudeThreshold: "1.0", HighFrequencyThreshold: "0.05"},
				},
			},
		},
	}
}

func ResourceToPromQueryExpr(namespace string, name string, resourceName *corev1.ResourceName) string {
	switch *resourceName {
	case corev1.ResourceCPU:
		return fmt.Sprintf(config.WorkloadCpuUsagePromQLFmtStr, namespace, name, "1m")
	case corev1.ResourceMemory:
		return fmt.Sprintf(config.WorkloadMemUsagePromQLFmtStr, namespace, name)
	}

	return ""
}
