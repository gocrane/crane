package utils

import (
	"github.com/gocrane/crane/pkg/common"
	"testing"
	"time"
)

func TestDetectTimestampCompletion(t *testing.T) {
	now := time.Now()
	timestamp := now.Unix()
	tests := []struct {
		name          string
		tsList        []*common.TimeSeries
		historyLength string
		want          bool
	}{
		{
			tsList: []*common.TimeSeries{
				{
					Samples: []common.Sample{
						{Timestamp: timestamp}, {Timestamp: now.Add(-time.Hour * 24).Unix()}, {Timestamp: now.Add(-time.Hour * 24 * 2).Unix()}, {Timestamp: now.Add(-time.Hour * 24 * 3).Unix()}, {Timestamp: now.Add(-time.Hour * 24 * 4).Unix()}, {Timestamp: now.Add(-time.Hour * 24 * 5).Unix()}, {Timestamp: now.Add(-time.Hour * 24 * 6).Unix()},
					},
				},
			},
			historyLength: "7d",
			want:          true,
		},
		{
			tsList: []*common.TimeSeries{
				{
					Samples: []common.Sample{
						{Timestamp: timestamp}, {Timestamp: now.Add(-time.Hour * 24).Unix()}, {Timestamp: now.Add(-time.Hour * 24 * 2).Unix()}, {Timestamp: now.Add(-time.Hour * 24 * 3).Unix()}, {Timestamp: now.Add(-time.Hour * 24 * 4).Unix()}, {Timestamp: now.Add(-time.Hour * 24 * 5).Unix()}, {Timestamp: now.Add(-time.Hour * 24 * 6).Unix()},
					},
				},
			},
			historyLength: "9d",
			want:          false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := DetectTimestampCompletion(tt.tsList, tt.historyLength, now)
			if got != tt.want {
				t.Errorf("DetectTimestampCompletion() = %v, want %v", got, tt.want)
			}
		})
	}
}
