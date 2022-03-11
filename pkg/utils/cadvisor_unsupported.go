//go:build !linux && !windows
// +build !linux,!windows

package utils

import (
	cmanager "github.com/google/cadvisor/manager"
	corelisters "k8s.io/client-go/listers/core/v1"
)

type CadvisorProvider struct {
	Manager   cmanager.Manager
	podLister corelisters.PodLister
}
