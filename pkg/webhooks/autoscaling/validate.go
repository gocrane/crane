package autoscaling

import (
	"context"
	"fmt"
	"time"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	"github.com/robfig/cron/v3"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/gocrane/crane/pkg/metricprovider"
)

type ValidationAdmission struct {
}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (p *ValidationAdmission) Default(ctx context.Context, req runtime.Object) error {
	return nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (p *ValidationAdmission) ValidateCreate(ctx context.Context, req runtime.Object) error {
	ehpa, ok := req.(*autoscalingapi.EffectiveHorizontalPodAutoscaler)
	if ok {
		if len(ehpa.Spec.Crons) > 0 {
			err := ValidateCronSpecs(ehpa)
			return err
		}
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (p *ValidationAdmission) ValidateUpdate(ctx context.Context, old, new runtime.Object) error {
	ehpa, ok := new.(*autoscalingapi.EffectiveHorizontalPodAutoscaler)
	if ok {
		if len(ehpa.Spec.Crons) > 0 {
			err := ValidateCronSpecs(ehpa)
			return err
		}
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (p *ValidationAdmission) ValidateDelete(ctx context.Context, req runtime.Object) error {
	return nil
}

func ValidateCronSpecs(ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler) error {
	if len(ehpa.Spec.Crons) > 0 {
		cronMetricNames := sets.NewString()
		cronNames := sets.NewString()
		for _, cronSpec := range ehpa.Spec.Crons {
			if cronSpec.Start == "" {
				return fmt.Errorf("cron start must not be empty")
			}
			if cronSpec.End == "" {
				return fmt.Errorf("cron end must not be empty")
			}
			if cronSpec.TimeZone != nil {
				_, err := time.LoadLocation(*cronSpec.TimeZone)
				if err != nil {
					return fmt.Errorf("cron timezone %v is not valid, please check the timezone format and make sure `$GOROOT/lib/time/zoneinfo.zip` in your server or image", *cronSpec.TimeZone)
				}
			}
			_, err := cron.ParseStandard(cronSpec.Start)
			if err != nil {
				return fmt.Errorf("cron %v start schedule parse failed: %v. err: %v", cronSpec.Name, cronSpec.Start, err)
			}
			_, err = cron.ParseStandard(cronSpec.End)
			if err != nil {
				return fmt.Errorf("cron %v end schedule parse failed: %v. err: %v", cronSpec.Name, cronSpec.End, err)
			}

			if cronNames.Has(cronSpec.Name) {
				return fmt.Errorf("cron name %v is duplicated", cronSpec.Name)
			}
			cronNames.Insert(cronSpec.Name)

			cronMetricName := metricprovider.EHPACronMetricName(ehpa.Namespace, ehpa.Name, cronSpec)
			if cronMetricNames.Has(cronMetricName) {
				return fmt.Errorf("constructed cron metric name %v is duplicated for cron %v, please check each cron name, timezone, start, end in ehpa cron spec, all characters will be transformed to lower case", cronMetricName, cronSpec.Name)
			}
			cronMetricNames.Insert(cronMetricName)
		}
	}
	return nil
}
