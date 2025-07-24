package metricprovider

import (
	"context"
	"testing"
	"time"
)

func TestCronTrigger(t *testing.T) {
	testCases := []struct {
		desc       string
		trigger    *CronTrigger
		now        time.Time
		wantActive bool
		wantErr    error
	}{
		{
			desc: "tc1",
			trigger: &CronTrigger{
				Name:     "cron-1",
				Location: time.Local,
				Start:    "15 10 ? * *",
				End:      "15 14 ? * *",
			},
			now:        time.Date(2022, 2, 17, 17, 0, 0, 0, time.Local),
			wantActive: false,
			wantErr:    nil,
		},
		{
			desc: "tc2",
			trigger: &CronTrigger{
				Name:     "cron-2",
				Location: time.Local,
				Start:    "15 10 ? * *",
				End:      "15 14 ? * *",
			},
			now:        time.Date(2022, 2, 17, 10, 15, 0, 0, time.Local),
			wantActive: true,
			wantErr:    nil,
		},
		{
			desc: "tc3",
			trigger: &CronTrigger{
				Name:     "cron-3",
				Location: time.Local,
				Start:    "15 10 ? * *",
				End:      "15 14 ? * *",
			},
			now:        time.Date(2022, 2, 17, 14, 15, 0, 0, time.Local),
			wantActive: true,
			wantErr:    nil,
		},
		{
			desc: "tc4",
			trigger: &CronTrigger{
				Name:     "cron-3",
				Location: time.Local,
				Start:    "15 10 ? * *",
				End:      "15 14 ? * *",
			},
			now:        time.Date(2022, 2, 17, 13, 15, 0, 0, time.Local),
			wantActive: true,
			wantErr:    nil,
		},
		{
			desc: "tc5",
			trigger: &CronTrigger{
				Name:     "cron-5",
				Location: time.Local,
				Start:    "15 10 ? * *",
				End:      "15 14 ? * *",
			},
			now:        time.Date(2022, 2, 17, 14, 15, 1, 0, time.Local),
			wantActive: false,
			wantErr:    nil,
		},
	}

	for _, tc := range testCases {
		gotActive, gotErr := tc.trigger.IsActive(context.TODO(), tc.now)
		if gotActive != tc.wantActive && gotErr != tc.wantErr {
			t.Fatalf("test case %v failed, wantActive %v, gotActive %v; wantErr: %v, gotErr: %v", tc.desc, tc.wantActive, gotActive, tc.wantErr, gotErr)
		}
	}
}
