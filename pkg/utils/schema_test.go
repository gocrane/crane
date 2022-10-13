package utils

import (
	"errors"
	"testing"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type SingluarRestMapper struct {
	meta.RESTMapper
}

func (mapper SingluarRestMapper) ResourceSingularizer(resource string) (singular string, err error) {
	return "", errors.New("raise a ResourceSingularizer error")
}

type MockRestMapper struct {
	meta.RESTMapper
}

func (mapper MockRestMapper) ResourceSingularizer(resource string) (singular string, err error) {
	return "apps.v1.pod", nil
}

func (mapper MockRestMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "pod"}, nil
}

type MockRestMapper2 struct {
	meta.RESTMapper
}

func (mapper MockRestMapper2) ResourceSingularizer(resource string) (singular string, err error) {
	return "apps.pod", nil
}

func (mapper MockRestMapper2) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{Group: "apps", Version: resource.Version, Kind: "pod"}, nil
}

func TestKindForResource(t *testing.T) {
	tests := []struct {
		name       string
		resource   string
		restMapper meta.RESTMapper
		want       string
		wantErr    bool
	}{
		{
			name:       "base",
			resource:   "pod",
			restMapper: SingluarRestMapper{},
			want:       "",
			wantErr:    true,
		},
		{
			name:       "fullySpecifiedGVR is not nil",
			resource:   "pod",
			restMapper: MockRestMapper{},
			want:       "pod",
			wantErr:    false,
		},
		{
			name:       "fullySpecifiedGVR is nil",
			resource:   "pod",
			restMapper: MockRestMapper2{},
			want:       "pod",
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := KindForResource(tt.resource, tt.restMapper)
			if (err != nil) != tt.wantErr {
				t.Errorf("KindForResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("KindForResource() = %v, want %v", got, tt.want)
			}
		})
	}
}
