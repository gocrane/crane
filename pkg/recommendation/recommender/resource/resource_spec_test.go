package resource

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
)

func TestGetResourceSpecifications(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []Specification
		wantErr bool
	}{
		{
			name:  "specs",
			input: "2c4g,4c8g,2c5g,2c1g,0.25c0.25g,0.5c1g,4c16g",
			want: []Specification{
				{
					CPU:    0.25,
					Memory: 0.25,
				},
				{
					CPU:    0.5,
					Memory: 1,
				},
				{
					CPU:    2,
					Memory: 1,
				},
				{
					CPU:    2,
					Memory: 4,
				},
				{
					CPU:    2,
					Memory: 5,
				},
				{
					CPU:    4,
					Memory: 8,
				},
				{
					CPU:    4,
					Memory: 16,
				},
			},
		},
		{
			name:    "specs format error1",
			input:   "24c",
			wantErr: true,
		},
		{
			name:    "specs format error2",
			input:   "2c4g5c",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetResourceSpecifications(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("GetResourceSpecifications not return error ")
			}
			if !reflect.DeepEqual(tt.want, got) {
				t.Errorf("GetResourceSpecifications %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetNormalizedResource(t *testing.T) {
	tests := []struct {
		name     string
		inputCpu string
		inputMem string
		wantCpu  string
		wantMem  string
	}{
		{
			name:     "specs1",
			inputCpu: "100m",
			inputMem: "125Mi",
			wantCpu:  "0.25",
			wantMem:  "256Mi",
		},
		{
			name:     "specs2",
			inputCpu: "100m",
			inputMem: "325Mi",
			wantCpu:  "0.25",
			wantMem:  "512Mi",
		},
		{
			name:     "specs3",
			inputCpu: "500m",
			inputMem: "1625Mi",
			wantCpu:  "1",
			wantMem:  "2Gi",
		},
	}

	specifications, _ := GetResourceSpecifications(DefaultSpecs)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpu := resource.MustParse(tt.inputCpu)
			mem := resource.MustParse(tt.inputMem)
			normalizedCpu, normalizedMem := GetNormalizedResource(&cpu, &mem, specifications)
			wantCpuQ := resource.MustParse(tt.wantCpu)
			wantMemQ := resource.MustParse(tt.wantMem)

			if normalizedCpu.Value() != wantCpuQ.Value() {
				t.Errorf("got cpu %s, want %s", normalizedCpu.String(), wantCpuQ.String())
			}

			if normalizedMem.Value() != wantMemQ.Value() {
				t.Errorf("got memory %s, want %s", normalizedMem.String(), wantMemQ.String())
			}
		})
	}
}
