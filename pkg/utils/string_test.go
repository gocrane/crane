package utils

import "testing"

func TestParseFloat(t *testing.T) {
	tests := []struct {
		name         string
		str          string
		defaultValue float64
		want         float64
		wantErr      bool
	}{
		// TODO: Add test cases.
		{
			name:         "empty string",
			str:          "",
			defaultValue: 0.16,
			want:         0.16,
			wantErr:      false,
		},
		{
			name:         "non empty string",
			str:          "0.64",
			defaultValue: 0.16,
			want:         0.64,
			wantErr:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFloat(tt.str, tt.defaultValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFloat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParsePercentage(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{
			name:    "empty string",
			input:   "",
			want:    0.00,
			wantErr: false,
		},
		{
			name:    "string parse error",
			input:   "1a",
			want:    0.00,
			wantErr: true,
		},
		{
			name:    "string parse ok",
			input:   "10%",
			want:    0.10,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePercentage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePercentage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParsePercentage() = %v, want %v", got, tt.want)
			}
		})
	}
}
