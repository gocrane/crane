package recommendation

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"

	analysisv1alph1 "github.com/gocrane/api/analysis/v1alpha1"
)

var (
	DefaultPeriodSeconds = int64(86400)
)

type ValidationAdmission struct {
}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (p *ValidationAdmission) Default(ctx context.Context, req runtime.Object) error {
	recommendation, ok := req.(*analysisv1alph1.Recommendation)
	if !ok {
		return fmt.Errorf("failed to convert req to Recommendation. ")
	}

	Default(recommendation)
	return nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (p *ValidationAdmission) ValidateCreate(ctx context.Context, req runtime.Object) error {
	recommendation, ok := req.(*analysisv1alph1.Recommendation)
	if !ok {
		return fmt.Errorf("failed to convert req to Recommendation. ")
	}

	klog.V(4).Info("validate create object %s", klog.KObj(recommendation))
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

func Default(recommendation *analysisv1alph1.Recommendation) {
	if recommendation.Spec.CompletionStrategy.CompletionStrategyType == "" {
		recommendation.Spec.CompletionStrategy.CompletionStrategyType = analysisv1alph1.CompletionStrategyOnce
	}

	if recommendation.Spec.CompletionStrategy.PeriodSeconds == nil {
		recommendation.Spec.CompletionStrategy.PeriodSeconds = &DefaultPeriodSeconds
	}
}
