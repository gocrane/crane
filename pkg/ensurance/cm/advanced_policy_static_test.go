package cm

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_advancedStaticPolicy_guaranteedCPUs(t *testing.T) {
	type args struct {
		pod       *v1.Pod
		container *v1.Container
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "pod guaranteedCPUs",
			args: args{
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							CPUSetAnnotation: string(CPUSetShare),
						},
					},
				},
				&v1.Container{
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("3"),
							v1.ResourceMemory: resource.MustParse("1G"),
						},
						Limits: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("3"),
							v1.ResourceMemory: resource.MustParse("1G"),
						},
					},
				},
			},
			want: 3,
		},
		{
			name: "pod guaranteedCPUs",
			args: args{
				&v1.Pod{},
				&v1.Container{
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("3"),
							v1.ResourceMemory: resource.MustParse("1G"),
						},
						Limits: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("3"),
							v1.ResourceMemory: resource.MustParse("1G"),
						},
					},
				},
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &advancedStaticPolicy{}
			if got := p.guaranteedCPUs(tt.args.pod, tt.args.container); got != tt.want {
				t.Errorf("advancedStaticPolicy.guaranteedCPUs() = %v, want %v", got, tt.want)
			}
		})
	}
}
