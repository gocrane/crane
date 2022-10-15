package utils

import (
	"reflect"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestCgroupName_ToCgroupfs(t *testing.T) {
	want := "/memory/memory.failcnt"
	got := CgroupName{"memory", "memory.failcnt"}.ToCgroupfs()
	if got != want {
		t.Errorf("CgroupName.ToCgroupfs() = %v, want %v", got, want)
	}
}

func TestGetCgroupPath(t *testing.T) {
	tests := []struct {
		name         string
		p            *v1.Pod
		cgroupDriver string
		want         string
	}{
		{
			name:         "base",
			p:            &v1.Pod{},
			cgroupDriver: "systemd",
			want:         "/",
		},
		{
			name:         "cgroupDriver is cgroupfs",
			p:            &v1.Pod{},
			cgroupDriver: "cgroupfs",
			want:         "/",
		},
		{
			name:         "cgroupDriver is default",
			p:            &v1.Pod{},
			cgroupDriver: "default",
			want:         "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCgroupPath(tt.p, tt.cgroupDriver); got != tt.want {
				t.Errorf("GetCgroupPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCgroupName(t *testing.T) {
	tests := []struct {
		name string
		p    *v1.Pod
		want CgroupName
	}{
		{
			name: "base",
			p:    &v1.Pod{},
			want: CgroupName{},
		},
		{
			name: "status is equal to PodQOSGuaranteed",
			p: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID: types.UID("fake-uid"),
				},
				Status: v1.PodStatus{QOSClass: v1.PodQOSGuaranteed},
			},
			want: NewCgroupName(RootCgroupName, CgroupKubePods, GetPodCgroupNameSuffix(types.UID("fake-uid"))),
		},
		{
			name: "status is equal to PodQOSBurstable",
			p: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID: types.UID("fake-uid"),
				},
				Status: v1.PodStatus{QOSClass: v1.PodQOSBurstable},
			},
			want: NewCgroupName(RootCgroupName, CgroupKubePods, strings.ToLower(string(v1.PodQOSBurstable)), GetPodCgroupNameSuffix(types.UID("fake-uid"))),
		},
		{
			name: "status is equal to PodQOSBestEffort",
			p: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID: types.UID("fake-uid"),
				},
				Status: v1.PodStatus{QOSClass: v1.PodQOSBestEffort},
			},
			want: NewCgroupName(RootCgroupName, CgroupKubePods, strings.ToLower(string(v1.PodQOSBestEffort)), GetPodCgroupNameSuffix(types.UID("fake-uid"))),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.want)
			if got := GetCgroupName(tt.p); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCgroupName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPodCgroupNameSuffix(t *testing.T) {
	want := "podfake-uid"
	got := GetPodCgroupNameSuffix(types.UID("fake-uid"))
	if got != want {
		t.Errorf("GetPodCgroupNameSuffix() = %v, want %v", got, want)
	}
}

func TestNewCgroupName(t *testing.T) {
	want := CgroupName{"cpu", "memory"}
	got := NewCgroupName(CgroupName{"cpu"}, []string{"memory"}...)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NewCgroupName() = %v, want %v", got, want)
	}
}

func TestCgroupName_ToSystemd(t *testing.T) {
	tests := []struct {
		name       string
		cgroupName CgroupName
		want       string
	}{
		{
			name:       "base",
			cgroupName: GetCgroupName(&v1.Pod{}),
			want:       "/",
		},
		{
			name: "len(slice) < len(suffix)",
			cgroupName: GetCgroupName(&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID: types.UID("fake-uid"),
				},
				Status: v1.PodStatus{QOSClass: v1.PodQOSGuaranteed},
			}), //[kubepods podfake-uid]
			want: "/kubepods.slice/kubepods-podfake_uid.slice",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cgroupName.ToSystemd(); got != tt.want {
				t.Errorf("CgroupName.ToSystemd() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_escapeSystemdCgroupName(t *testing.T) {
	want := "cpu_memory"
	got := escapeSystemdCgroupName("cpu-memory")
	if got != want {
		t.Errorf("escapeSystemdCgroupName() = %v, want %v", got, want)
	}
}

func TestExpandSlice(t *testing.T) {
	tests := []struct {
		name    string
		slice   string
		want    string
		wantErr bool
	}{
		{
			name:    "base",
			slice:   "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "slice contains '/'",
			slice:   "/.slice",
			want:    "",
			wantErr: true,
		},
		{
			name:    "slice contains '-'",
			slice:   "-.slice",
			want:    "/",
			wantErr: false,
		},
		{
			name:    "slice contains many '-'",
			slice:   "--.slice",
			want:    "",
			wantErr: true,
		},
		{
			name:    "slice contains some contents and many '-'",
			slice:   "cpu-memory.slice",
			want:    "/cpu.slice/cpu-memory.slice",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandSlice(tt.slice)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandSlice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExpandSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}
