package podgroupprediction

import (
	"context"
	"net/http"

	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/gocrane/api/prediction/v1alpha1"
)

// ValidatingAdmission validates cluster object when creating/updating/deleting.
type ValidatingAdmission struct {
	decoder *admission.Decoder
}

// Check if our ValidatingAdmission implements necessary interface
var _ admission.Handler = &ValidatingAdmission{}
var _ admission.DecoderInjector = &ValidatingAdmission{}

// Handle implements admission.Handler interface.
// It yields a response to an AdmissionRequest.
func (v *ValidatingAdmission) Handle(ctx context.Context, req admission.Request) admission.Response {
	pgPrediction := &v1alpha1.PodGroupPrediction{}

	err := v.decoder.Decode(req, pgPrediction)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	klog.V(2).Infof("Validating cluster(%s) for request: %s", pgPrediction.Name, req.Operation)

	spec := pgPrediction.Spec
	if spec.Mode == v1alpha1.PredictionModeRange {
		for _, mc := range spec.MetricPredictionConfigs {
			if mc.DSP == nil {
				return admission.Denied("only dsp supports PredictionModeRange, percentile only supports instant")
			}
		}

	}
	if len(spec.Pods) == 0 && spec.WorkloadRef == nil && len(spec.LabelSelector.MatchExpressions) == 0 && len(spec.LabelSelector.MatchLabels) == 0 {
		return admission.Denied("Pods, WorkloadRef, LabelSelector must supplied at least one")
	}
	return admission.Allowed("")
}

// InjectDecoder implements admission.DecoderInjector interface.
// A decoder will be automatically injected.
func (v *ValidatingAdmission) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}
