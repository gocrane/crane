package recommendation

import (
	"context"
	"time"

	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	analysisv1alpha1 "github.com/gocrane/api/analysis/v1alpha1"

	"github.com/gocrane/crane/pkg/metrics"
)

type Checker struct {
	client.Client
	MonitorInterval time.Duration
	OutDateInterval time.Duration
}

func (r Checker) Run(stopCh <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(r.MonitorInterval)
		defer ticker.Stop()

		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				r.runChecker()
			}
		}
	}()
}

func (r Checker) runChecker() {
	recommendList := &analysisv1alpha1.RecommendationList{}
	err := r.Client.List(context.TODO(), recommendList, []client.ListOption{}...)
	if err != nil {
		klog.Errorf("Failed to list recommendation: %v", err)
	}

	for _, recommend := range recommendList.Items {
		updateStatus := "Updated"
		if time.Now().Sub(recommend.Status.LastUpdateTime.Time) > r.OutDateInterval {
			updateStatus = "OutDate"
		}

		resultStatus := "Failed"
		if len(recommend.Status.RecommendedInfo) != 0 || len(recommend.Status.RecommendedValue) != 0 {
			resultStatus = "Success"
		}

		metrics.RecommendationsStatus.With(map[string]string{
			"type":          string(recommend.Spec.Type),
			"apiversion":    recommend.Spec.TargetRef.APIVersion,
			"owner_kind":    recommend.Spec.TargetRef.Kind,
			"namespace":     recommend.Spec.TargetRef.Namespace,
			"owner_name":    recommend.Spec.TargetRef.Name,
			"update_status": updateStatus,
			"result_status": resultStatus,
		}).Set(time.Now().Sub(recommend.Status.LastUpdateTime.Time).Seconds())
	}
}
