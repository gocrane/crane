package cm

import (
	"testing"
)

func TestAdvancedCpuManager_loadKubeletPolicy(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		want     string
	}{
		{
			name:     "file not exist",
			fileName: "/not-exist",
			want:     "",
		},
		{
			name:     "file get none",
			fileName: "./test/" + cpuManagerStateFileName,
			want:     "none",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &AdvancedCpuManager{}
			if got := m.loadKubeletPolicy(tt.fileName); got != tt.want {
				t.Errorf("AdvancedCpuManager.loadKubeletPolicy() = %v, want %v", got, tt.want)
			}
		})
	}
}
