package idlenode

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/providers"
	"github.com/gocrane/crane/pkg/recommendation/framework"
	"github.com/gocrane/crane/pkg/utils"
)

const callerFormat = "IdleNodeRecommender-%s-%s"

// CheckDataProviders in PrePrepare phase, will create data source provider via your recommendation config.
func (inr *IdleNodeRecommender) CheckDataProviders(ctx *framework.RecommendationContext) error {
	if err := inr.BaseRecommender.CheckDataProviders(ctx); err != nil {
		return err
	}

	return nil
}

func (inr *IdleNodeRecommender) CollectData(ctx *framework.RecommendationContext) error {
	labelSelector := labels.SelectorFromSet(ctx.Identity.Labels)
	caller := fmt.Sprintf(callerFormat, klog.KObj(ctx.Recommendation), ctx.Recommendation.UID)
	timeNow := time.Now()
	if inr.cpuUsageUtilization > 0 {
		metricNamer := metricnaming.ResourceToGeneralMetricNamer(utils.GetNodeCpuUsageUtilizationExpression(ctx.Recommendation.Spec.TargetRef.Name), corev1.ResourceCPU, labelSelector, caller)
		if err := metricNamer.Validate(); err != nil {
			return err
		}
		ctx.MetricNamer = metricNamer

		// get node cpu usage utilization
		klog.Infof("%s: %s CpuQuery %s", ctx.String(), inr.Name(), ctx.MetricNamer.BuildUniqueKey())
		tsList, err := ctx.DataProviders[providers.PrometheusDataSource].QueryTimeSeries(ctx.MetricNamer, timeNow.Add(-time.Hour*24*7), timeNow, time.Minute)
		if err != nil {
			return fmt.Errorf("%s query node cpu usage historic metrics failed: %v ", inr.Name(), err)
		}
		if len(tsList) != 1 {
			return fmt.Errorf("%s query node cpu usage historic metrics data is unexpected, List length is %d ", inr.Name(), len(tsList))
		}
		ctx.AddInputValue(cpuUsageUtilizationKey, tsList)
	}

	if inr.memoryUsageUtilization > 0 {
		metricNamer := metricnaming.ResourceToGeneralMetricNamer(utils.GetNodeMemUsageUtilizationExpression(ctx.Recommendation.Spec.TargetRef.Name), corev1.ResourceMemory, labelSelector, caller)
		if err := metricNamer.Validate(); err != nil {
			return err
		}
		// get node memory usage utilization
		klog.Infof("%s: %s MemoryQuery %s", ctx.String(), inr.Name(), metricNamer.BuildUniqueKey())
		tsList, err := ctx.DataProviders[providers.PrometheusDataSource].QueryTimeSeries(metricNamer, timeNow.Add(-time.Hour*24*7), timeNow, time.Minute)
		if err != nil {
			return fmt.Errorf("%s query node memory usage historic metrics failed: %v ", inr.Name(), err)
		}
		if len(tsList) != 1 {
			return fmt.Errorf("%s query node memory usage historic metrics data is unexpected, List length is %d ", inr.Name(), len(tsList))
		}
		ctx.AddInputValue(memoryUsageUtilizationKey, tsList)
	}

	if inr.cpuRequestUtilization > 0 {
		metricNamer := metricnaming.ResourceToGeneralMetricNamer(utils.GetNodeCpuRequestUtilizationExpression(ctx.Recommendation.Spec.TargetRef.Name), corev1.ResourceCPU, labelSelector, caller)
		if err := metricNamer.Validate(); err != nil {
			return err
		}
		ctx.MetricNamer = metricNamer

		// get node cpu request utilization
		klog.Infof("%s: %s CpuQuery %s", ctx.String(), inr.Name(), metricNamer)
		tsList, err := ctx.DataProviders[providers.PrometheusDataSource].QueryTimeSeries(metricNamer, timeNow.Add(-time.Hour*24*7), timeNow, time.Minute)
		if err != nil {
			return fmt.Errorf("%s query node cpu request historic metrics failed: %v ", inr.Name(), err)
		}
		if len(tsList) != 1 {
			return fmt.Errorf("%s query node cpu request historic metrics data is unexpected, List length is %d ", inr.Name(), len(tsList))
		}
		ctx.AddInputValue(cpuRequestUtilizationKey, tsList)
	}

	if inr.memoryRequestUtilization > 0 {
		metricNamer := metricnaming.ResourceToGeneralMetricNamer(utils.GetNodeMemRequestUtilizationExpression(ctx.Recommendation.Spec.TargetRef.Name), corev1.ResourceMemory, labelSelector, caller)
		if err := metricNamer.Validate(); err != nil {
			return err
		}

		// get node memory request utilization
		klog.Infof("%s: %s MemoryQuery %s", ctx.String(), inr.Name(), metricNamer.BuildUniqueKey())
		tsList, err := ctx.DataProviders[providers.PrometheusDataSource].QueryTimeSeries(metricNamer, timeNow.Add(-time.Hour*24*7), timeNow, time.Minute)
		if err != nil {
			return fmt.Errorf("%s query node memory request historic metrics failed: %v ", inr.Name(), err)
		}
		if len(tsList) != 1 {
			return fmt.Errorf("%s query node memory request historic metrics data is unexpected, List length is %d ", inr.Name(), len(tsList))
		}
		ctx.AddInputValue(memoryRequestUtilizationKey, tsList)
	}

	return nil
}

func (inr *IdleNodeRecommender) PostProcessing(ctx *framework.RecommendationContext) error {
	return nil
}
