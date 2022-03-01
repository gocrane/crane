package ensurance

import (
	"fmt"

	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	genericvalidation "k8s.io/apimachinery/pkg/api/validation"
	metavalidation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/kubernetes/pkg/apis/core"

	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
	"github.com/gocrane/crane/pkg/ensurance/collector"
	"github.com/gocrane/crane/pkg/known"
)

type NepValidationAdmission struct {
}

type ActionValidationAdmission struct {
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (p *NepValidationAdmission) ValidateCreate(ctx context.Context, req runtime.Object) error {

	nep, ok := req.(*ensuranceapi.NodeQOSEnsurancePolicy)
	if !ok {
		return fmt.Errorf("req can not convert to NodeQOSEnsurancePolicy")
	}

	allErrs := genericvalidation.ValidateObjectMeta(&nep.ObjectMeta, false, genericvalidation.NameIsDNS1035Label, field.NewPath("metadata"))

	if nep.Spec.Selector != nil {
		allErrs = append(allErrs, metavalidation.ValidateLabelSelector(nep.Spec.Selector, field.NewPath("spec").Child("selector"))...)
	}

	allErrs = append(allErrs, validateNodeQualityProbe(nep.Spec.NodeQualityProbe, field.NewPath("nodeQualityProbe"))...)

	var httpGetEnable bool
	if nep.Spec.NodeQualityProbe.HTTPGet != nil {
		httpGetEnable = true
	}
	allErrs = append(allErrs, validateObjectiveEnsurances(nep.Spec.ObjectiveEnsurances, field.NewPath("objectiveEnsurances"), httpGetEnable)...)

	if len(allErrs) != 0 {
		return allErrs.ToAggregate()
	}

	return nil
}

func validateNodeQualityProbe(nodeProbe ensuranceapi.NodeQualityProbe, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	if (nodeProbe.NodeLocalGet == nil) && (nodeProbe.HTTPGet == nil) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("httpGet"), "", "HttpGet and nodeLocalGet cannot be empty at the same time."))
		return allErrs
	}

	if (nodeProbe.NodeLocalGet != nil) && (nodeProbe.HTTPGet != nil) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("httpGet"), "", "The httpGet and nodeLocalGet can not set at the same time"))
		return allErrs
	}

	if nodeProbe.HTTPGet != nil {
		allErrs = append(allErrs, validateHTTPGetAction(nodeProbe.HTTPGet, fldPath.Child("httpGet"))...)
	}

	return allErrs
}

func validateHTTPGetAction(http *corev1.HTTPGetAction, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(http.Path) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("path"), ""))
	}

	allErrs = append(allErrs, validatePortNumOrName(http.Port, fldPath.Child("port"))...)

	var supportedHTTPSchemes = sets.NewString(string(core.URISchemeHTTP), string(core.URISchemeHTTPS))
	if !supportedHTTPSchemes.Has(string(http.Scheme)) {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("scheme"), http.Scheme, supportedHTTPSchemes.List()))
	}

	for _, header := range http.HTTPHeaders {
		for _, msg := range validation.IsHTTPHeaderName(header.Name) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("httpHeaders"), header.Name, msg))
		}
	}

	return allErrs
}

func validateObjectiveEnsurances(objects []ensuranceapi.ObjectiveEnsurance, fldPath *field.Path, httpGetEnable bool) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(objects) == 0 {
		return append(allErrs, field.Required(fldPath, ""))
	}

	allNames := sets.String{}
	for i, obj := range objects {
		idxPath := fldPath.Index(i)

		if obj.Name == "" {
			allErrs = append(allErrs, field.Required(idxPath.Child("name"), ""))
		} else {
			for _, msg := range validation.IsDNS1123Label(obj.Name) {
				allErrs = append(allErrs, field.Invalid(idxPath.Child("name"), obj.Name, msg))
			}
		}

		if allNames.Has(obj.Name) {
			allErrs = append(allErrs, field.Duplicate(idxPath.Child("name"), obj.Name))
		} else {
			allNames.Insert(obj.Name)
		}

		//check actionName
		if obj.AvoidanceActionName == "" {
			allErrs = append(allErrs, field.Required(idxPath.Child("actionName"), ""))
		} else {
			for _, msg := range validation.IsDNS1123Label(obj.AvoidanceActionName) {
				allErrs = append(allErrs, field.Invalid(idxPath.Child("actionName"), obj.AvoidanceActionName, msg))
			}
		}

		if obj.AvoidanceThreshold != nil {
			allErrs = append(allErrs, genericvalidation.ValidateNonnegativeField(int64(*obj.AvoidanceThreshold), idxPath.Child("avoidanceThreshold"))...)
		}

		if obj.RestoreThreshold != nil {
			allErrs = append(allErrs, genericvalidation.ValidateNonnegativeField(int64(*obj.RestoreThreshold), idxPath.Child("restoreThreshold"))...)
		}

		if obj.MetricRule == nil {
			allErrs = append(allErrs, field.Required(idxPath.Child("metricRule"), ""))
		} else {
			allErrs = append(allErrs, validateMetricRule(obj.MetricRule, idxPath.Child("metricRule"), httpGetEnable)...)
		}
	}

	return allErrs
}

func validateMetricRule(rule *ensuranceapi.MetricRule, fldPath *field.Path, httpGetEnable bool) field.ErrorList {
	allErrs := field.ErrorList{}

	if rule.Name == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), ""))
	} else {
		if !httpGetEnable {
			if !collector.CheckMetricNameExist(rule.Name) {
				allErrs = append(allErrs, field.NotSupported(fldPath.Child("name"), rule.Name, []string{}))
			}
		}
	}

	if rule.Selector != nil {
		allErrs = append(allErrs, metavalidation.ValidateLabelSelector(rule.Selector, fldPath.Child("selector"))...)
	}

	if rule.Value.Cmp(resource.Quantity{}) <= 0 {
		allErrs = append(allErrs, field.Invalid(fldPath, rule.Value.String(), genericvalidation.IsNegativeErrorMsg))
	}

	return allErrs
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (p *NepValidationAdmission) ValidateUpdate(ctx context.Context, old, new runtime.Object) error {

	nepOld, ok := old.(*ensuranceapi.NodeQOSEnsurancePolicy)
	if !ok {
		return fmt.Errorf("old can not convert to NodeQOSEnsurancePolicy")
	}

	nep, ok := old.(*ensuranceapi.NodeQOSEnsurancePolicy)
	if !ok {
		return fmt.Errorf("new can not convert to NodeQOSEnsurancePolicy")
	}

	allErrs := genericvalidation.ValidateObjectMetaUpdate(&nep.ObjectMeta, &nepOld.ObjectMeta, field.NewPath("metadata"))

	if len(allErrs) != 0 {
		return allErrs.ToAggregate()
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (p *NepValidationAdmission) ValidateDelete(ctx context.Context, req runtime.Object) error {
	return nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (p *ActionValidationAdmission) ValidateCreate(ctx context.Context, req runtime.Object) error {
	action, ok := req.(*ensuranceapi.AvoidanceAction)
	if !ok {
		return fmt.Errorf("req can not convert to AvoidanceAction")
	}

	allErrs := genericvalidation.ValidateObjectMeta(&action.ObjectMeta, false, genericvalidation.NameIsDNSLabel, field.NewPath("metadata"))
	allErrs = append(allErrs, validateAvoidanceActionSpec(action.Spec, field.NewPath("spec"))...)

	if len(allErrs) != 0 {
		return allErrs.ToAggregate()
	}

	return nil
}

func validateAvoidanceActionSpec(spec ensuranceapi.AvoidanceActionSpec, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	if spec.CoolDownSeconds != nil {
		allErrs = append(allErrs, genericvalidation.ValidateNonnegativeField(int64(*spec.CoolDownSeconds), fldPath.Child("coolDownSeconds"))...)
	}

	if spec.Throttle != nil {
		allErrs = append(allErrs, validateThrottleAction(spec.Throttle, fldPath.Child("throttle"))...)
	}

	if spec.Eviction != nil {
		allErrs = append(allErrs, validateEvictionAction(spec.Eviction, fldPath.Child("eviction"))...)
	}

	return allErrs
}

func validateThrottleAction(throttle *ensuranceapi.ThrottleAction, fldPath *field.Path) field.ErrorList {

	var allErrs field.ErrorList

	allErrs = append(allErrs, genericvalidation.ValidateNonnegativeField(int64(throttle.CPUThrottle.MinCPURatio), fldPath.Child("cpuThrottle").Child("minCPURatio"))...)
	allErrs = append(allErrs, genericvalidation.ValidateNonnegativeField(int64(throttle.CPUThrottle.StepCPURatio), fldPath.Child("cpuThrottle").Child("stepCPURatio"))...)

	if throttle.CPUThrottle.MinCPURatio > known.MaxMinCPURatio {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("cpuThrottle").Child("minCPURatio"), throttle.CPUThrottle.MinCPURatio, fmt.Sprintf("must be lesser than or equal to %d", known.MaxMinCPURatio)))
	}

	if throttle.CPUThrottle.StepCPURatio > known.MaxStepCPURatio {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("cpuThrottle").Child("stepCPURatio"), throttle.CPUThrottle.StepCPURatio, fmt.Sprintf("must be lesser than or equal to %d", known.MaxStepCPURatio)))
	}

	return allErrs
}

func validateEvictionAction(eviction *ensuranceapi.EvictionAction, fldPath *field.Path) field.ErrorList {

	var allErrs field.ErrorList

	if eviction.TerminationGracePeriodSeconds != nil {
		allErrs = append(allErrs, genericvalidation.ValidateNonnegativeField(int64(*eviction.TerminationGracePeriodSeconds), fldPath.Child("terminationGracePeriodSeconds"))...)
	}

	return allErrs
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (p *ActionValidationAdmission) ValidateUpdate(ctx context.Context, old, new runtime.Object) error {
	nepOld, ok := old.(*ensuranceapi.AvoidanceAction)
	if !ok {
		return fmt.Errorf("old can not convert to AvoidanceAction")
	}

	nep, ok := old.(*ensuranceapi.AvoidanceAction)
	if !ok {
		return fmt.Errorf("new can not convert to AvoidanceAction")
	}

	allErrs := genericvalidation.ValidateObjectMetaUpdate(&nep.ObjectMeta, &nepOld.ObjectMeta, field.NewPath("metadata"))

	if len(allErrs) != 0 {
		return allErrs.ToAggregate()
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (p *ActionValidationAdmission) ValidateDelete(ctx context.Context, req runtime.Object) error {
	return nil
}

//Copied from k8s.io/kubernetes/pkg/apis/core/validation/validation.go
func validatePortNumOrName(port intstr.IntOrString, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if port.Type == intstr.Int {
		for _, msg := range validation.IsValidPortNum(port.IntValue()) {
			allErrs = append(allErrs, field.Invalid(fldPath, port.IntValue(), msg))
		}
	} else if port.Type == intstr.String {
		for _, msg := range validation.IsValidPortName(port.StrVal) {
			allErrs = append(allErrs, field.Invalid(fldPath, port.StrVal, msg))
		}
	} else {
		allErrs = append(allErrs, field.InternalError(fldPath, fmt.Errorf("unknown type: %v", port.Type)))
	}
	return allErrs
}
