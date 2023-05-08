package service

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/providers"
	"github.com/gocrane/crane/pkg/recommendation/framework"
	"github.com/gocrane/crane/pkg/utils"
)

const callerFormat = "ServiceRecommender-%s-%s"

// CheckDataProviders in PrePrepare phase, will create data source provider via your recommendation config.
func (s *ServiceRecommender) CheckDataProviders(ctx *framework.RecommendationContext) error {
	if err := s.BaseRecommender.CheckDataProviders(ctx); err != nil {
		return err
	}

	return nil
}

func (s *ServiceRecommender) CollectData(ctx *framework.RecommendationContext) error {
	if len(ctx.Pods) == 0 {
		return nil
	}

	var workloadRef *metav1.OwnerReference
	for _, pod := range ctx.Pods {
		workloadRef = utils.GetPodOwnerReference(ctx.Context, ctx.Client, &pod)
		if workloadRef != nil {
			break
		}
	}
	if workloadRef == nil {
		return fmt.Errorf("could not find all pod OwnerReferences for Service %s selector", ctx.Object.GetName())
	}
	podName := utils.GetPodNameReg(workloadRef.Name, workloadRef.Kind)

	labelSelector := labels.SelectorFromSet(ctx.Identity.Labels)
	caller := fmt.Sprintf(callerFormat, klog.KObj(ctx.Recommendation), ctx.Recommendation.UID)
	timeNow := time.Now()
	metricNamer := metricnaming.ResourceToGeneralMetricNamer(utils.GetWorkloadNetReceiveBytesExpression(podName), corev1.ResourceServices, labelSelector, caller)
	if err := metricNamer.Validate(); err != nil {
		return err
	}
	ctx.MetricNamer = metricNamer

	// get pod net receive bytes
	klog.Infof("%s: %s NetReceiveBytes %s", ctx.String(), s.Name(), ctx.MetricNamer.BuildUniqueKey())
	tsList, err := ctx.DataProviders[providers.PrometheusDataSource].QueryTimeSeries(ctx.MetricNamer, timeNow.Add(-time.Hour*24*7), timeNow, time.Minute)
	if err != nil {
		return fmt.Errorf("%s query pod net receive bytes historic metrics failed: %v ", s.Name(), err)
	}
	if len(tsList) != 1 {
		return fmt.Errorf("%s query pod net receive bytes historic metrics data is unexpected, List length is %d ", s.Name(), len(tsList))
	}
	ctx.AddInputValue(netReceiveBytesKey, tsList)

	metricNamer = metricnaming.ResourceToGeneralMetricNamer(utils.GetWorkloadNetTransferBytesExpression(podName), corev1.ResourceServices, labelSelector, caller)
	if err = metricNamer.Validate(); err != nil {
		return err
	}

	// get pod net transfer bytes
	klog.Infof("%s: %s NetTransferBytes %s", ctx.String(), s.Name(), ctx.MetricNamer.BuildUniqueKey())
	tsList, err = ctx.DataProviders[providers.PrometheusDataSource].QueryTimeSeries(ctx.MetricNamer, timeNow.Add(-time.Hour*24*7), timeNow, time.Minute)
	if err != nil {
		return fmt.Errorf("%s query pod net transfer bytes historic metrics failed: %v ", s.Name(), err)
	}
	if len(tsList) != 1 {
		return fmt.Errorf("%s query pod net transfer bytes historic metrics data is unexpected, List length is %d ", s.Name(), len(tsList))
	}
	ctx.AddInputValue(netTransferBytesKey, tsList)

	return nil
}

func (s *ServiceRecommender) PostProcessing(ctx *framework.RecommendationContext) error {
	return nil
}
