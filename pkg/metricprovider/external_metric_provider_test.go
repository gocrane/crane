package metricprovider

import (
	"reflect"
	"testing"
	"time"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
)

func StringPtr(str string) *string {
	return &str
}

func TestGetCronScaleLocation(t *testing.T) {
	americaLoc, _ := time.LoadLocation("America/Adak")
	asiaShanghaiLoc, _ := time.LoadLocation("Asia/Shanghai")
	testCases := []struct {
		desc      string
		cronScale autoscalingapi.CronSpec
		want      *time.Location
	}{
		{
			desc: "tc1.",
			cronScale: autoscalingapi.CronSpec{
				TimeZone: StringPtr("Local"),
			},
			want: time.Local,
		},
		{
			desc:      "tc2. null timezone is UTC",
			cronScale: autoscalingapi.CronSpec{},
			want:      time.UTC,
		},
		{
			desc: "tc3.",
			cronScale: autoscalingapi.CronSpec{
				TimeZone: StringPtr("America/Adak"),
			},
			want: americaLoc,
		},
		{
			desc: "tc4. non null timezone but unknown loc is UTC",
			cronScale: autoscalingapi.CronSpec{
				TimeZone: StringPtr("unknown"),
			},
			want: time.UTC,
		},
		{
			desc: "tc5. Asia/Shanghai",
			cronScale: autoscalingapi.CronSpec{
				TimeZone: StringPtr("Asia/Shanghai"),
			},
			want: asiaShanghaiLoc,
		},
	}

	for _, tc := range testCases {
		gotLoc := GetCronScaleLocation(tc.cronScale)
		if !reflect.DeepEqual(gotLoc.String(), tc.want.String()) {
			t.Fatalf("test case %v failed, wantLoc: %v, gotLoc: %v", tc.desc, tc.want, gotLoc)
		}
	}
}

func TestEHPACronMetricName(t *testing.T) {
	testCases := []struct {
		desc      string
		namespace string
		name      string
		cronScale autoscalingapi.CronSpec
		wantName  string
	}{
		{
			desc:      "tc1.",
			namespace: "default",
			name:      "ehpa",
			cronScale: autoscalingapi.CronSpec{
				Name:     "cron1",
				TimeZone: StringPtr("Local"),
				Start:    "15 10 ? * *",
				End:      "15 14 ? * *",
			},
			wantName: "cron-default-ehpa-cron1-local-1510qmxx-1514qmxx",
		},
		{
			desc:      "tc2. null timezone is Local",
			namespace: "default",
			name:      "ehpa",
			cronScale: autoscalingapi.CronSpec{
				Name:  "",
				Start: "15 10 ? * *",
				End:   "15 14 ? * *",
			},
			wantName: "cron-default-ehpa--utc-1510qmxx-1514qmxx",
		},
		{
			desc:      "tc3.",
			namespace: "default",
			name:      "ehpa",
			cronScale: autoscalingapi.CronSpec{
				Name:     "cron3",
				TimeZone: StringPtr("America/Adak"),
				Start:    "15 10 ? * *",
				End:      "15 14 ? * *",
			},
			wantName: "cron-default-ehpa-cron3-america-adak-1510qmxx-1514qmxx",
		},
		{
			desc:      "tc4. non-null timezone but unknown loc is UTC",
			namespace: "default",
			name:      "ehpa",
			cronScale: autoscalingapi.CronSpec{
				Name:     "cron4",
				TimeZone: StringPtr("unknown"),
				Start:    "15 10 ? * *",
				End:      "15 14 ? * *",
			},
			wantName: "cron-default-ehpa-cron4-utc-1510qmxx-1514qmxx",
		},
	}

	for _, tc := range testCases {
		gotName := EHPACronMetricName(tc.namespace, tc.name, tc.cronScale)
		if !reflect.DeepEqual(gotName, tc.wantName) {
			t.Fatalf("test case %v failed, wantName: %v, gotName: %v", tc.desc, tc.wantName, gotName)
		}
	}
}
