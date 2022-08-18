package replicas

import (
	"fmt"
	"math"
	"time"

	"github.com/montanaflynn/stats"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/recommend/types"
	"github.com/gocrane/crane/pkg/recommendation/framework"
	"github.com/gocrane/crane/pkg/utils"
)

func (rr *ReplicasRecommender) PreRecommend(ctx *framework.RecommendationContext) error {
	// we load algorithm config in this phase
	// TODO(chrisydxie) support configuration
	config := &config.Config{
		DSP: &predictionapi.DSP{
			SampleInterval: "1m",
			HistoryLength:  "7d",
			Estimators:     predictionapi.Estimators{},
		},
	}
	ctx.AlgorithmConfig = config
	return nil
}

func (rr *ReplicasRecommender) Recommend(ctx *framework.RecommendationContext) error {
	p := ctx.PredictorMgr.GetPredictor(predictionapi.AlgorithmTypeDSP)
	timeNow := time.Now()
	caller := fmt.Sprintf(rr.Name(), klog.KObj(&ctx.RecommendationRule), ctx.RecommendationRule.UID)

	// get workload cpu usage
	tsListPrediction, err := utils.QueryPredictedTimeSeriesOnce(p, caller,
		ctx.AlgorithmConfig,
		ctx.MetricNamer,
		timeNow,
		timeNow.Add(time.Hour*24*7))

	if err != nil {
		klog.Warningf("%s query predicted time series failed: %v ", rr.Name(), err)
	}

	if len(tsListPrediction) != 1 {
		klog.Warningf("%s prediction metrics data is unexpected, List length is %d ", rr.Name(), len(tsListPrediction))
	}

	ctx.ResultValues = tsListPrediction
	return nil
}

// Policy add some logic for result of recommend phase.
func (rr *ReplicasRecommender) Policy(ctx *framework.RecommendationContext) error {
	// get max value of history and predicted data
	var cpuMax float64
	var cpuUsages []float64
	// combine values from historic and prediction
	for _, sample := range ctx.InputValues[0].Samples {
		cpuUsages = append(cpuUsages, sample.Value)
		if sample.Value > cpuMax {
			cpuMax = sample.Value
		}
	}

	if len(ctx.ResultValues) >= 1 {
		for _, sample := range ctx.ResultValues[0].Samples {
			cpuUsages = append(cpuUsages, sample.Value)
			if sample.Value > cpuMax {
				cpuMax = sample.Value
			}
		}
	}

	// apply policy for predicted values
	percentileCpu, err := stats.Percentile(cpuUsages, rr.CpuPercentile)
	if err != nil {
		return fmt.Errorf("%s get percentileCpu failed: %v", rr.Name(), err)
	}

	requestTotal, err := utils.CalculatePodTemplateRequests(&ctx.PodTemplate, corev1.ResourceCPU)
	if err != nil {
		return fmt.Errorf("%s CalculatePodTemplateRequests failed: %v", rr.Name(), err)
	}

	minReplicas, err := rr.ProposeMinReplicas(percentileCpu, requestTotal)
	if err != nil {
		return fmt.Errorf("%s proposeMinReplicas failed: %v", rr.Name(), err)
	}

	replicasRecommendation := &types.ReplicasRecommendation{
		Replicas: &minReplicas,
	}

	result := types.ProposedRecommendation{
		ReplicasRecommendation: replicasRecommendation,
	}

	resultBytes, err := yaml.Marshal(result)
	if err != nil {
		return fmt.Errorf("%s proposeMinReplicas failed: %v", rr.Name(), err)
	}

	ctx.Recommendation.Status.RecommendedValue = string(resultBytes)

	unstructed := ctx.Object.(*unstructured.Unstructured)
	newUnstructed := unstructed.DeepCopy()
	err = unstructured.SetNestedField(newUnstructed.Object, int64(minReplicas), "spec", "replicas")
	if err != nil {
		return fmt.Errorf("set replicas to spec failed %s. ", err)
	}

	newPatch, oldPatch, err := framework.ConvertToRecommendationInfos(unstructed.Object, newUnstructed.Object)
	if err != nil {
		return fmt.Errorf("convert to recommendation infos failed: %s. ", err)
	}

	ctx.Recommendation.Status.RecommendedInfo = string(newPatch)
	ctx.Recommendation.Status.CurrentInfo = string(oldPatch)
	if ctx.Scale.Spec.Replicas == minReplicas {
		ctx.Recommendation.Status.Action = "None"
	} else {
		ctx.Recommendation.Status.Action = "Patch"
	}

	return nil
}

// ProposeMinReplicas calculate min replicas based on default-min-replicas
func (rr *ReplicasRecommender) ProposeMinReplicas(workloadCpu float64, requestTotal int64) (int32, error) {
	minReplicas := int32(rr.DefaultMinReplicas)

	// minReplicas should be larger than 0
	if minReplicas < 1 {
		minReplicas = 1
	}

	min := int32(math.Ceil(workloadCpu / (float64(rr.TargetUtilization) / 100. * float64(requestTotal) / 1000.)))
	if min > minReplicas {
		minReplicas = min
	}

	return minReplicas, nil
}
