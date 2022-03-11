//go:build !linux && !windows
// +build !linux,!windows

package utils

import (
	"errors"
	cmanager "github.com/google/cadvisor/manager"
	v1 "k8s.io/api/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	statsapi "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
)

var errUnsupported = errors.New("cAdvisor is unsupported in this build")

type CadvisorProvider struct {
	Manager   cmanager.Manager
	podLister corelisters.PodLister
}

func NewCadvisorProvider(manager cmanager.Manager, podLister corelisters.PodLister) *CadvisorProvider {
	return &CadvisorProvider{
		Manager:   manager,
		podLister: podLister,
	}
}
func (c *CadvisorProvider) GetCPUAndMemoryStats() (*statsapi.Summary, error) {
	return &statsapi.Summary{}, errUnsupported
}

func GetContainerNameFromPod(pod *v1.Pod, containerId string) string {
	return ""
}

func GetCgroupPath(p *v1.Pod) string {
	return ""
}
