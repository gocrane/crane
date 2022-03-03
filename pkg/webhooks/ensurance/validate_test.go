package ensurance

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/validation/field"

	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
	"github.com/gocrane/crane/pkg/known"
)

func TestValidateNodeQualityProbe(t *testing.T) {
	cases := map[string]struct {
		nodeProbe   ensuranceapi.NodeQualityProbe
		expectErr   bool
		errorType   field.ErrorType
		errorDetail string
	}{
		"invalid NodeQualityProbe, httpGet and nodeLocalGet not set": {
			nodeProbe:   ensuranceapi.NodeQualityProbe{},
			errorType:   field.ErrorTypeInvalid,
			errorDetail: "HttpGet and nodeLocalGet cannot be empty at the same time.",
			expectErr:   true,
		},
		"valid NodeQualityProbe, node local set ": {
			nodeProbe: ensuranceapi.NodeQualityProbe{
				NodeLocalGet: &ensuranceapi.NodeLocalGet{
					LocalCacheTTLSeconds: 60},
			},
			expectErr: false,
		},
		"invalid NodeQualityProbe, httpGet and nodeLocalGet can not set at the same time": {
			nodeProbe: ensuranceapi.NodeQualityProbe{
				NodeLocalGet: &ensuranceapi.NodeLocalGet{
					LocalCacheTTLSeconds: 60},
				HTTPGet: &corev1.HTTPGetAction{
					Host: "127.0.0.1",
				},
			},
			errorType:   field.ErrorTypeInvalid,
			errorDetail: "The httpGet and nodeLocalGet can not set at the same time",
			expectErr:   true,
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			errs := validateNodeQualityProbe(v.nodeProbe, field.NewPath("nodeQualityProbe"))
			t.Logf("%s: len %d", k, len(errs))
			if v.expectErr && len(errs) > 0 {
				if errs[0].Type != v.errorType || !strings.Contains(errs[0].Detail, v.errorDetail) {
					t.Errorf("[%s] Expected error type %q with detail %q, got %v", k, v.errorType, v.errorDetail, errs)
				}
			} else if v.expectErr && len(errs) == 0 {
				t.Errorf("Unexpected success")
			}
			if !v.expectErr && len(errs) != 0 {
				t.Errorf("Unexpected error(s): %v", errs)
			}
		})
	}
}

func TestValidateMetricRule(t *testing.T) {
	cases := map[string]struct {
		rule          ensuranceapi.MetricRule
		httpGetEnable bool
		expectErr     bool
		errorType     field.ErrorType
		errorDetail   string
	}{
		"metric name is required": {
			rule:          ensuranceapi.MetricRule{},
			httpGetEnable: false,
			errorType:     field.ErrorTypeRequired,
			errorDetail:   "",
			expectErr:     true,
		},
		"metric name is no support": {
			rule:          ensuranceapi.MetricRule{Name: "not_support_metircs"},
			httpGetEnable: false,
			errorType:     field.ErrorTypeNotSupported,
			errorDetail:   "",
			expectErr:     true,
		},
		"value is invalid, value is empty": {
			rule:          ensuranceapi.MetricRule{Name: "cpu_total_usage"},
			httpGetEnable: false,
			errorType:     field.ErrorTypeInvalid,
			errorDetail:   "",
			expectErr:     true,
		},
		"value is invalid, value is negative": {
			rule: ensuranceapi.MetricRule{
				Name:  "cpu_total_usage",
				Value: resource.MustParse("-1")},
			httpGetEnable: false,
			errorType:     field.ErrorTypeInvalid,
			errorDetail:   "",
			expectErr:     true,
		},
		"value is valid": {
			rule: ensuranceapi.MetricRule{
				Name:  "cpu_total_usage",
				Value: resource.MustParse("6000")},
			httpGetEnable: false,
			expectErr:     false,
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			errs := validateMetricRule(&v.rule, field.NewPath("metricRule"), v.httpGetEnable)
			t.Logf("%s: len %d", k, len(errs))
			if v.expectErr && len(errs) > 0 {
				if errs[0].Type != v.errorType || !strings.Contains(errs[0].Detail, v.errorDetail) {
					t.Errorf("[%s] Expected error type %q with detail %q, got %v", k, v.errorType, v.errorDetail, errs)
				}
			} else if v.expectErr && len(errs) == 0 {
				t.Errorf("Unexpected success")
			}
			if !v.expectErr && len(errs) != 0 {
				t.Errorf("Unexpected error(s): %v", errs)
			}
		})
	}
}

func TestValidateObjectiveEnsurances(t *testing.T) {
	cases := map[string]struct {
		objects       []ensuranceapi.ObjectiveEnsurance
		httpGetEnable bool
		expectErr     bool
		errorType     field.ErrorType
		errorDetail   string
	}{
		"object is required": {
			objects:       []ensuranceapi.ObjectiveEnsurance{},
			httpGetEnable: false,
			errorType:     field.ErrorTypeRequired,
			errorDetail:   "",
			expectErr:     true,
		},
		"object name is  required": {
			objects:       []ensuranceapi.ObjectiveEnsurance{{Name: ""}},
			httpGetEnable: false,
			errorType:     field.ErrorTypeRequired,
			errorDetail:   "",
			expectErr:     true,
		},
		"object name is invalid": {
			objects:       []ensuranceapi.ObjectiveEnsurance{{Name: "aaa.bbb"}},
			httpGetEnable: false,
			errorType:     field.ErrorTypeInvalid,
			errorDetail:   "",
			expectErr:     true,
		},
		"object name is duplicate": {
			objects: []ensuranceapi.ObjectiveEnsurance{{Name: "aaa",
				AvoidanceActionName: "eviction",
				AvoidanceThreshold:  known.DefaultAvoidedThreshold,
				RestoreThreshold:    known.DefaultRestoredThreshold,
				MetricRule: &ensuranceapi.MetricRule{
					Name:  "cpu_total_usage",
					Value: resource.MustParse("6000")}},
				{Name: "aaa"}},
			httpGetEnable: false,
			errorType:     field.ErrorTypeDuplicate,
			errorDetail:   "",
			expectErr:     true,
		},
		"object action name is  required": {
			objects:       []ensuranceapi.ObjectiveEnsurance{{Name: "aaa"}},
			httpGetEnable: false,
			errorType:     field.ErrorTypeRequired,
			errorDetail:   "",
			expectErr:     true,
		},
		"object action name is invalid": {
			objects:       []ensuranceapi.ObjectiveEnsurance{{Name: "aaa", AvoidanceActionName: "eviction.aa"}},
			httpGetEnable: false,
			errorType:     field.ErrorTypeInvalid,
			errorDetail:   "",
			expectErr:     true,
		},
		"object metric rule is required": {
			objects: []ensuranceapi.ObjectiveEnsurance{{Name: "aaa", AvoidanceActionName: "eviction",
				AvoidanceThreshold: known.DefaultAvoidedThreshold, RestoreThreshold: known.DefaultRestoredThreshold}},
			httpGetEnable: false,
			errorType:     field.ErrorTypeRequired,
			errorDetail:   "",
			expectErr:     true,
		},
		"object valid": {
			objects: []ensuranceapi.ObjectiveEnsurance{{Name: "aaa",
				AvoidanceActionName: "eviction",
				AvoidanceThreshold:  known.DefaultAvoidedThreshold,
				RestoreThreshold:    known.DefaultRestoredThreshold,
				MetricRule: &ensuranceapi.MetricRule{
					Name:  "cpu_total_usage",
					Value: resource.MustParse("6000")}}},
			httpGetEnable: false,
			expectErr:     false,
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			errs := validateObjectiveEnsurances(v.objects, field.NewPath("objectiveEnsurances"), v.httpGetEnable)
			t.Logf("%s: len %d", k, len(errs))
			if v.expectErr && len(errs) > 0 {
				if errs[0].Type != v.errorType || !strings.Contains(errs[0].Detail, v.errorDetail) {
					t.Errorf("[%s] Expected error type %q with detail %q, got %v", k, v.errorType, v.errorDetail, errs)
				}
			} else if v.expectErr && len(errs) == 0 {
				t.Errorf("Unexpected success")
			}
			if !v.expectErr && len(errs) != 0 {
				t.Errorf("Unexpected error(s): %v", errs)
			}
		})
	}
}
