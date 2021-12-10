// +build !windows

package runtime

import (
	"os"
	"syscall"
)

var defaultRuntimeEndpoints = []string{"unix:///var/run/dockershim.sock", "unix:///run/containerd/containerd.sock", "unix:///run/crio/crio.sock"}

var shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}
