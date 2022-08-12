package replicas

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/montanaflynn/stats"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/common"
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

	for _, sample := range ctx.ResultValues[0].Samples {
		cpuUsages = append(cpuUsages, sample.Value)
		if sample.Value > cpuMax {
			cpuMax = sample.Value
		}
	}

	// get percentile value
	cpuPercentile, err := strconv.ParseFloat(rr.Config["replicas.cpu-percentile"], 64)
	if err != nil {
		return fmt.Errorf("%s parse replicas.cpu-percentile failed: %v", rr.Name(), err)
	}

	// apply policy for predicted values
	percentileCpu, err := stats.Percentile(cpuUsages, cpuPercentile)
	if err != nil {
		return fmt.Errorf("%s get percentileCpu failed: %v", rr.Name(), err)
	}

	err = rr.checkMinCpuUsageThreshold(cpuMax)
	if err != nil {
		return fmt.Errorf("%s checkMinCpuUsageThreshold failed: %v", rr.Name(), err)
	}

	medianMin, medianMax, err := rr.minMaxMedians(ctx.InputValues)
	if err != nil {
		return fmt.Errorf("%s minMaxMedians failed: %v", rr.Name(), err)
	}

	err = rr.checkFluctuation(medianMin, medianMax)
	if err != nil {
		return fmt.Errorf("%s checkFluctuation failed: %v", rr.Name(), err)
	}

	requestTotal, err := utils.CalculatePodTemplateRequests(ctx.PodTemplate, corev1.ResourceCPU)
	if err != nil {
		return fmt.Errorf("%s CalculatePodTemplateRequests failed: %v", rr.Name(), err)
	}

	minReplicas, err := rr.proposeMinReplicas(percentileCpu, requestTotal)
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

	return nil
}

// checkMinCpuUsageThreshold check if the max cpu for target is reach to replicas.min-cpu-usage-threshold
func (rr *ReplicasRecommender) checkMinCpuUsageThreshold(cpuMax float64) error {
	minCpuUsageThreshold, err := strconv.ParseFloat(rr.Config["replicas.min-cpu-usage-threshold"], 64)
	if err != nil {
		return err
	}

	klog.V(4).Infof("%s checkMinCpuUsageThreshold, cpuMax %f threshold %f", rr.Name(), cpuMax, minCpuUsageThreshold)
	if cpuMax < minCpuUsageThreshold {
		return fmt.Errorf("target cpuusage %f is under replicas.min-cpu-usage-threshold %f. ", cpuMax, minCpuUsageThreshold)
	}

	return nil
}

func (rr *ReplicasRecommender) minMaxMedians(predictionTs []*common.TimeSeries) (float64, float64, error) {
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
func (rr *ReplicasRecommender) checkFluctuation(medianMin, medianMax float64) error {
	fluctuationThreshold, err := strconv.ParseFloat(rr.Config["replicas.fluctuation-threshold"], 64)
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

// proposeMinReplicas calculate min replicas based on replicas.default-min-replicas
func (rr *ReplicasRecommender) proposeMinReplicas(workloadCpu float64, requestTotal int64) (int32, error) {
	defaultMinReplicas, err := strconv.ParseInt(rr.Config["replicas.default-min-replicas"], 10, 32)
	if err != nil {
		return 0, err
	}

	targetUtilization, err := strconv.ParseInt(rr.Config["replicas.cpu-target-utilization"], 10, 32)
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
