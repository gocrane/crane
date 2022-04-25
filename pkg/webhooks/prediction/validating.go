package prediction

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
)

type ValidationAdmission struct {
}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (p *ValidationAdmission) Default(ctx context.Context, req runtime.Object) error {
	return nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (p *ValidationAdmission) ValidateCreate(ctx context.Context, req runtime.Object) error {
	tsp, ok := req.(*predictionapi.TimeSeriesPrediction)
	if ok {
		if tsp.Spec.TargetRef.Name == "" {
			return fmt.Errorf("need TargetRef.Name")
		}
		if tsp.Spec.PredictionWindowSeconds < 3600 {
			return fmt.Errorf("PredictionWindowSeconds at least 3600")
		}

		if len(tsp.Spec.PredictionMetrics) == 0 {
			return fmt.Errorf("PredictionMetrics is null")
		}

		sets := sets.NewString()
		for _, pm := range tsp.Spec.PredictionMetrics {
			sets.Insert(pm.ResourceIdentifier)
		}
		if sets.Len() < len(tsp.Spec.PredictionMetrics) {
			return fmt.Errorf("PredictionMetrics has duplicated metric, each resourceIdentifier must be unique")
		}
		for _, pm := range tsp.Spec.PredictionMetrics {

			if pm.Type == predictionapi.ResourceQueryMetricType && pm.ResourceQuery == nil {
				return fmt.Errorf("PredictionMetric type is %v, but no query specified", pm.Type)
			}
			if pm.Type == predictionapi.MetricQueryMetricType && pm.MetricQuery == nil {
				return fmt.Errorf("PredictionMetric type is %v, but no query specified", pm.Type)
			}
			if pm.Type == predictionapi.ExpressionQueryMetricType && pm.ExpressionQuery == nil {
				return fmt.Errorf("PredictionMetric type is %v, but no query specified", pm.Type)
			}

		}
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (p *ValidationAdmission) ValidateUpdate(ctx context.Context, old, new runtime.Object) error {
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (p *ValidationAdmission) ValidateDelete(ctx context.Context, req runtime.Object) error {
	return nil
}
