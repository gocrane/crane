package utils

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		want    time.Duration
		wantErr bool
	}{
		{
			name:    "duration string's time unit is 'd'",
			s:       "1d",
			want:    time.Hour * 24 * time.Duration(1),
			wantErr: false,
		},
		{
			name:    "duration string's time unit is not 'd'",
			s:       "300ms",
			want:    300 * time.Millisecond,
			wantErr: false,
		},
		{
			name:    "duration string's format is invalid",
			s:       "",
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDuration(tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name    string
		ts      string
		want    time.Time
		wantErr bool
	}{
		{
			name:    "parsed incorrect",
			ts:      "",
			wantErr: true,
		},
		{
			name:    "parsed correct",
			ts:      "1d",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseDuration(tt.ts)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
