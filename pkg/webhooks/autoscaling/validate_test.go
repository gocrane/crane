package autoscaling

import (
	"fmt"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"

	"github.com/gocrane/crane/pkg/utils"
)

func TestValidateCronSpecs(t *testing.T) {
	Start1 := "00 00 ? * *"
	End1 := "00 06 ? * *"

	want3 := fmt.Errorf("cron timezone %v is not valid, please check the timezone format and make sure `$GOROOT/lib/time/zoneinfo.zip` in your server or image", "XXX")

	want4 := fmt.Errorf("cron start must not be empty")
	want5 := fmt.Errorf("cron end must not be empty")

	want6 := fmt.Errorf("cron name %v is duplicated", "cron6")

	want7 := fmt.Errorf("constructed cron metric name %v is duplicated for cron %v, please check each cron name, timezone, start, end in ehpa cron spec, all characters will be transformed to lower case", "cron-default-ehpa-cron7-local-0000qmxx-0006qmxx", "Cron7")

	testCases := []struct {
		desc string
		ehpa *autoscalingapi.EffectiveHorizontalPodAutoscaler
		want error
	}{
		{
			desc: "tc1.",
			ehpa: &autoscalingapi.EffectiveHorizontalPodAutoscaler{
				Spec: autoscalingapi.EffectiveHorizontalPodAutoscalerSpec{
					Crons: []autoscalingapi.CronSpec{
						{
							Name:     "cron1",
							TimeZone: utils.StringPtr("Local"),
							Start:    Start1,
							End:      End1,
						},
					},
				},
			},
		},
		{
			desc: "tc3.",
			ehpa: &autoscalingapi.EffectiveHorizontalPodAutoscaler{
				Spec: autoscalingapi.EffectiveHorizontalPodAutoscalerSpec{
					Crons: []autoscalingapi.CronSpec{
						{
							Name:     "cron3",
							TimeZone: utils.StringPtr("XXX"),
							Start:    "00 00 ? * *",
							End:      "00 06 ? * *",
						},
					},
				},
			},
			want: want3,
		},
		{
			desc: "tc4.",
			ehpa: &autoscalingapi.EffectiveHorizontalPodAutoscaler{
				Spec: autoscalingapi.EffectiveHorizontalPodAutoscalerSpec{
					Crons: []autoscalingapi.CronSpec{
						{
							Name:     "cron4",
							TimeZone: utils.StringPtr("Local"),
							End:      "00 06 ? * *",
						},
					},
				},
			},
			want: want4,
		},
		{
			desc: "tc5.",
			ehpa: &autoscalingapi.EffectiveHorizontalPodAutoscaler{
				Spec: autoscalingapi.EffectiveHorizontalPodAutoscalerSpec{
					Crons: []autoscalingapi.CronSpec{
						{
							Name:     "cron5",
							TimeZone: utils.StringPtr("Local"),
							Start:    "00 00 ? * *",
						},
					},
				},
			},
			want: want5,
		},
		{
			desc: "tc6.",
			ehpa: &autoscalingapi.EffectiveHorizontalPodAutoscaler{
				Spec: autoscalingapi.EffectiveHorizontalPodAutoscalerSpec{
					Crons: []autoscalingapi.CronSpec{
						{
							Name:     "cron6",
							TimeZone: utils.StringPtr("Local"),
							Start:    "00 00 ? * *",
							End:      "00 06 ? * *",
						},
						{
							Name:     "cron6",
							TimeZone: utils.StringPtr("Local"),
							Start:    "00 06 ? * *",
							End:      "00 09 ? * *",
						},
					},
				},
			},
			want: want6,
		},
		{
			desc: "tc7.",
			ehpa: &autoscalingapi.EffectiveHorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ehpa",
					Namespace: "default",
				},
				Spec: autoscalingapi.EffectiveHorizontalPodAutoscalerSpec{
					Crons: []autoscalingapi.CronSpec{
						{
							Name:     "cron7",
							TimeZone: utils.StringPtr("Local"),
							Start:    "00 00 ? * *",
							End:      "00 06 ? * *",
						},
						{
							Name:     "Cron7",
							TimeZone: utils.StringPtr("Local"),
							Start:    "00 00 ? * *",
							End:      "00 06 ? * *",
						},
					},
				},
			},
			want: want7,
		},
	}

	for _, tc := range testCases {
		gotErr := ValidateCronSpecs(tc.ehpa)
		if !reflect.DeepEqual(gotErr, tc.want) {
			t.Fatalf("tc %v failed, gotErr: %v, wantErr: %v", tc.desc, gotErr, tc.want)
		}
	}
}
