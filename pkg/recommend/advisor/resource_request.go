package advisor

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/metricquery"
	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/recommend/types"
	"github.com/gocrane/crane/pkg/utils"
)

const callerFormat = "RecommendationCaller-%s"

const (
	DefaultNamespace = "default"
)

type ResourceRequestAdvisor struct {
	*types.Context
}

func makeCpuConfig(props map[string]string) *config.Config {
	sampleInterval, exists := props["resource.cpu-sample-interval"]
	if !exists {
		sampleInterval = "1m"
	}
	percentile, exists := props["resource.cpu-request-percentile"]
	if !exists {
		percentile = "0.99"
	}
	marginFraction, exists := props["resource.cpu-request-margin-fraction"]
	if !exists {
		marginFraction = "0.15"
	}

	return &config.Config{
		Percentile: &predictionapi.Percentile{
			Aggregated:     true,
			SampleInterval: sampleInterval,
			MarginFraction: marginFraction,
			Percentile:     percentile,
			Histogram: predictionapi.HistogramConfig{
				HalfLife:   "24h",
				BucketSize: "0.1",
				MaxValue:   "100",
			},
		},
	}
}

func makeMemConfig(props map[string]string) *config.Config {
	sampleInterval, exists := props["resource.mem-sample-interval"]
	if !exists {
		sampleInterval = "1m"
	}
	percentile, exists := props["resource.mem-request-percentile"]
	if !exists {
		percentile = "0.99"
	}
	marginFraction, exists := props["resource.mem-request-margin-fraction"]
	if !exists {
		marginFraction = "0.15"
	}

	return &config.Config{
		Percentile: &predictionapi.Percentile{
			Aggregated:     true,
			SampleInterval: sampleInterval,
			MarginFraction: marginFraction,
			Percentile:     percentile,
			Histogram: predictionapi.HistogramConfig{
				HalfLife:   "48h",
				BucketSize: "104857600",
				MaxValue:   "104857600000",
			},
		},
	}
}

func (a *ResourceRequestAdvisor) Advise(proposed *types.ProposedRecommendation) error {
	r := &types.ResourceRequestRecommendation{}

	p := a.PredictorMgr.GetPredictor(predictionapi.AlgorithmTypePercentile)
	if p == nil {
		return fmt.Errorf("predictor %v not found", predictionapi.AlgorithmTypePercentile)
	}

	if len(a.Pods) == 0 {
		return fmt.Errorf("pod not found")
	}

	pod := a.Pods[0]
	namespace := pod.Namespace

	var queryExpr string
	for _, c := range pod.Spec.Containers {
		cr := types.ContainerRecommendation{
			ContainerName: c.Name,
			Target:        map[corev1.ResourceName]string{},
		}

		mericNamer := ResourceToContainerMetricNamer(namespace, a.Recommendation.Spec.TargetRef.Name, c.Name, corev1.ResourceCPU)
		klog.V(6).Infof("CPU query for resource request recommendation: %s", mericNamer.BuildUniqueKey())
		cpuConfig := makeCpuConfig(a.ConfigProperties)
		tsList, err := utils.QueryPredictedValuesOnce(a.Recommendation, p,
			fmt.Sprintf(callerFormat, a.Recommendation.UID), cpuConfig, mericNamer)
		if err != nil {
			return err
		}
		if len(tsList) < 1 || len(tsList[0].Samples) < 1 {
			return fmt.Errorf("no value retured for queryExpr: %s", queryExpr)
		}
		v := int64(tsList[0].Samples[0].Value * 1000)
		cr.Target[corev1.ResourceCPU] = resource.NewMilliQuantity(v, resource.DecimalSI).String()

		mericNamer = ResourceToContainerMetricNamer(namespace, a.Recommendation.Spec.TargetRef.Name, c.Name, corev1.ResourceMemory)
		klog.V(6).Infof("Memory query for resource request recommendation: %s", mericNamer.BuildUniqueKey())
		memConfig := makeMemConfig(a.ConfigProperties)
		tsList, err = utils.QueryPredictedValuesOnce(a.Recommendation, p,
			fmt.Sprintf(callerFormat, a.Recommendation.UID), memConfig, mericNamer)
		if err != nil {
			return err
		}
		if len(tsList) < 1 || len(tsList[0].Samples) < 1 {
			return fmt.Errorf("no value retured for queryExpr: %s", queryExpr)
		}
		v = int64(tsList[0].Samples[0].Value)
		cr.Target[corev1.ResourceMemory] = resource.NewQuantity(v, resource.BinarySI).String()

		r.Containers = append(r.Containers, cr)
	}

	proposed.ResourceRequest = r
	return nil
}

func (a *ResourceRequestAdvisor) Name() string {
	return "ResourceRequestAdvisor"
}

func ResourceToContainerMetricNamer(namespace, workloadname, containername string, resourceName corev1.ResourceName) metricnaming.MetricNamer {
	// container
	return &metricnaming.GeneralMetricNamer{
		Metric: &metricquery.Metric{
			Type:       metricquery.ContainerMetricType,
			MetricName: resourceName.String(),
			Container: &metricquery.ContainerNamerInfo{
				Namespace:     namespace,
				WorkloadName:  workloadname,
				ContainerName: containername,
				Selector:      labels.Everything(),
			},
		},
	}
}
