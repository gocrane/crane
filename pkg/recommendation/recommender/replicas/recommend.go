package replicas

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/montanaflynn/stats"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/recommend/types"
	"github.com/gocrane/crane/pkg/recommendation/framework"
	"github.com/gocrane/crane/pkg/utils"
)

type PatchReplicas struct {
	Spec PatchReplicasSpec `json:"spec,omitempty"`
}

type PatchReplicasSpec struct {
	Replicas *int32 `json:"replicas,omitempty"`
}

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
	caller := fmt.Sprintf(rr.Name(), klog.KObj(ctx.RecommendationRule), ctx.RecommendationRule.UID)

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
	minReplicas, _, _, err := rr.GetMinReplicas(ctx)
	if err != nil {
		return err
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

	var newPatch PatchReplicas
	newPatch.Spec.Replicas = &minReplicas
	newPatchBytes, err := json.Marshal(newPatch)
	if err != nil {
		return fmt.Errorf("marshal newPatch failed %s. ", err)
	}

	var oldPatch PatchReplicas
	oldPatch.Spec.Replicas = &ctx.Scale.Spec.Replicas
	oldPatchBytes, err := json.Marshal(oldPatch)
	if err != nil {
		return fmt.Errorf("marshal oldPatch failed %s. ", err)
	}

	ctx.Recommendation.Status.RecommendedInfo = string(newPatchBytes)
	ctx.Recommendation.Status.CurrentInfo = string(oldPatchBytes)
	ctx.Recommendation.Status.Action = "Patch"

	return nil
}

// ProposeMinReplicas calculate min replicas based on default-min-replicas
func (rr *ReplicasRecommender) ProposeMinReplicas(resourceUsage float64, requestTotal int64, targetUtilization float64) (int32, error) {
	minReplicas := int32(rr.DefaultMinReplicas)

	// minReplicas should be larger than 0
	if minReplicas < 1 {
		minReplicas = 1
	}

	min := int32(math.Ceil(resourceUsage / (targetUtilization * float64(requestTotal) / 1000.)))
	if min > minReplicas {
		minReplicas = min
	}

	return minReplicas, nil
}

func (rr *ReplicasRecommender) GetMinReplicas(ctx *framework.RecommendationContext) (int32, float64, float64, error) {
	var cpuUsages []float64
	var cpuMax float64
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
		return 0, 0, 0, fmt.Errorf("%s get percentileCpu failed: %v", rr.Name(), err)
	}

	requestTotalCpu, err := utils.CalculatePodTemplateRequests(&ctx.PodTemplate, corev1.ResourceCPU)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%s CalculatePodTemplateRequests cpu failed: %v", rr.Name(), err)
	}

	klog.Infof("%s: WorkloadCpuUsage Percentile %f PodCpuRequest %d CPUTargetUtilization %f", ctx.String(), percentileCpu, requestTotalCpu, rr.CPUTargetUtilization)
	minReplicasCpu, err := rr.ProposeMinReplicas(percentileCpu, requestTotalCpu, rr.CPUTargetUtilization)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%s proposeMinReplicas for cpu failed: %v", rr.Name(), err)
	}

	var memUsages []float64
	// combine values from historic and prediction
	for _, sample := range ctx.InputValues2[0].Samples {
		memUsages = append(memUsages, sample.Value)
	}

	percentileMem, err := stats.Percentile(memUsages, rr.MemPercentile)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%s get percentileMem failed: %v", rr.Name(), err)
	}

	requestTotalMem, err := utils.CalculatePodTemplateRequests(&ctx.PodTemplate, corev1.ResourceMemory)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%s CalculatePodTemplateRequests failed: %v", rr.Name(), err)
	}

	klog.Infof("%s: WorkloadMemoryUsage Percentile %f PodMemoryRequest %f MemTargetUtilization %f", ctx.String(), percentileMem, float64(requestTotalMem)/1000, rr.MemTargetUtilization)
	minReplicasMem, err := rr.ProposeMinReplicas(percentileMem, requestTotalMem, rr.MemTargetUtilization)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("%s proposeMinReplicas for cpu failed: %v", rr.Name(), err)
	}

	if minReplicasMem > minReplicasCpu {
		minReplicasCpu = minReplicasMem
	}

	return minReplicasCpu, cpuMax, percentileCpu, nil
}
