package replicas

import (
	"fmt"
	"time"

	"github.com/gocrane/crane/pkg/metricnaming"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/providers"
	"github.com/gocrane/crane/pkg/recommendation/framework"
)

const callerFormat = "ReplicasRecommendationCaller-%s-%s"

// CheckDataProviders in PrePrepare phase, will create data source provider via your recommendation config.
func (rr *ReplicasRecommender) CheckDataProviders(ctx *framework.RecommendationContext) error {
	if err := rr.BaseRecommender.CheckDataProviders(ctx); err != nil {
		return err
	}

	return nil
}

func (rr *ReplicasRecommender) CollectData(ctx *framework.RecommendationContext) error {
	resourceCpu := corev1.ResourceCPU
	labelSelector := labels.SelectorFromSet(ctx.Identity.Labels)
	caller := fmt.Sprintf(callerFormat, klog.KObj(ctx.Recommendation), ctx.Recommendation.UID)
	metricNamer := metricnaming.ResourceToWorkloadMetricNamer(ctx.Recommendation.Spec.TargetRef.DeepCopy(), &resourceCpu, labelSelector, caller)
	if err := metricNamer.Validate(); err != nil {
		return err
	}
	ctx.MetricNamer = metricNamer

	klog.V(4).Infof("%s CpuQuery %s RecommendationRule %s", rr.Name(), ctx.MetricNamer.BuildUniqueKey(), klog.KObj(&ctx.RecommendationRule))
	timeNow := time.Now()
	tsList, err := ctx.DataProviders[providers.PrometheusDataSource].QueryTimeSeries(ctx.MetricNamer, timeNow.Add(-time.Hour*24*7), timeNow, time.Minute)
	if err != nil {
		return fmt.Errorf("%s query historic metrics failed: %v ", rr.Name(), err)
	}
	if len(tsList) != 1 {
		return fmt.Errorf("%s query historic metrics data is unexpected, List length is %d ", rr.Name(), len(tsList))
	}
	ctx.InputValues = tsList
	return nil
}

func (rr *ReplicasRecommender) PostProcessing(ctx *framework.RecommendationContext) error {
	return nil
}
