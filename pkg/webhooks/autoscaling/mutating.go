package autoscaling

import (
	"context"
	"fmt"
	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
)

type MutatingAdmission struct {
}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (m *MutatingAdmission) Default(ctx context.Context, obj runtime.Object) error {
	ehpa, ok := obj.(*autoscalingapi.EffectiveHorizontalPodAutoscaler)
	if !ok {
		return fmt.Errorf("expected a ehpa but got a %T", obj)
	}
	klog.Infof("Into EHPA injection %s/%s", ehpa.Namespace, ehpa.Name)

	return nil
}
