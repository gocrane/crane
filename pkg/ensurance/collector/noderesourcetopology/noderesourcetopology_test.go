package noderesourcetopology

import (
	"testing"

	kubeletconfiginternal "k8s.io/kubernetes/pkg/kubelet/apis/config"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
)

func Test_parseReservedSystemCPUs(t *testing.T) {
	type args struct {
		kubeletConfig *kubeletconfiginternal.KubeletConfiguration
	}
	tests := []struct {
		name                   string
		args                   args
		want                   cpuset.CPUSet
		wantErr                bool
		wantSystemReservedCPUs string
	}{
		{
			name: "empty cpu",
			args: args{
				kubeletConfig: newKubeletConfig(nil, nil, "2-3"),
			},
			want:    cpuset.NewCPUSet(2, 3),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseReservedSystemCPUs(tt.args.kubeletConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseReservedSystemCPUs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !got.Equals(tt.want) {
				t.Errorf("parseReservedSystemCPUs() got = %v, want %v", got.String(), tt.want.String())
			}
		})
	}
}

func newKubeletConfig(systemReserved, kubeReserved map[string]string,
	reservedSystemCPUs string) *kubeletconfiginternal.KubeletConfiguration {
	return &kubeletconfiginternal.KubeletConfiguration{
		SystemReserved:     systemReserved,
		KubeReserved:       kubeReserved,
		ReservedSystemCPUs: reservedSystemCPUs,
	}
}
