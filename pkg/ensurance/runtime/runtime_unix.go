//go:build !windows
// +build !windows

package runtime

var defaultRuntimeEndpoints = []string{"unix:///var/run/dockershim.sock", "unix:///run/containerd/containerd.sock", "unix:///run/crio/crio.sock", "unix:///run/k3s/containerd/containerd.sock"}
