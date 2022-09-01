package utils

import (
	"fmt"
	"path"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (cgroupName CgroupName) ToCgroupfs() string {
	return "/" + path.Join(cgroupName...)
}

func GetCgroupPath(p *v1.Pod, cgroupDriver string) string {
	cgroupName := GetCgroupName(p)
	switch cgroupDriver {
	case "systemd":
		return cgroupName.ToSystemd()
	case "cgroupfs":
		return cgroupName.ToCgroupfs()
	default:
		return ""
	}
}

var RootCgroupName = CgroupName([]string{})

func GetCgroupName(p *v1.Pod) CgroupName {
	switch p.Status.QOSClass {
	case v1.PodQOSGuaranteed:
		return NewCgroupName(RootCgroupName, CgroupKubePods, GetPodCgroupNameSuffix(p.UID))
	case v1.PodQOSBurstable:
		return NewCgroupName(RootCgroupName, CgroupKubePods, strings.ToLower(string(v1.PodQOSBurstable)), GetPodCgroupNameSuffix(p.UID))
	case v1.PodQOSBestEffort:
		return NewCgroupName(RootCgroupName, CgroupKubePods, strings.ToLower(string(v1.PodQOSBestEffort)), GetPodCgroupNameSuffix(p.UID))
	default:
		return RootCgroupName
	}
}

const (
	podCgroupNamePrefix = "pod"
)

func GetPodCgroupNameSuffix(podUID types.UID) string {
	return podCgroupNamePrefix + string(podUID)
}

type CgroupName []string

func NewCgroupName(base CgroupName, components ...string) CgroupName {
	return append(append([]string{}, base...), components...)
}

// systemdSuffix is the cgroup name suffix for systemd
const systemdSuffix string = ".slice"

func (cgroupName CgroupName) ToSystemd() string {
	if len(cgroupName) == 0 || (len(cgroupName) == 1 && cgroupName[0] == "") {
		return "/"
	}
	newparts := []string{}
	for _, part := range cgroupName {
		part = escapeSystemdCgroupName(part)
		newparts = append(newparts, part)
	}

	result, err := ExpandSlice(strings.Join(newparts, "-") + systemdSuffix)
	if err != nil {
		// Should never happen...
		panic(fmt.Errorf("error converting cgroup name [%v] to systemd format: %v", cgroupName, err))
	}
	return result
}

func escapeSystemdCgroupName(part string) string {
	return strings.Replace(part, "-", "_", -1)
}

func ExpandSlice(slice string) (string, error) {
	suffix := ".slice"
	// Name has to end with ".slice", but can't be just ".slice".
	if len(slice) < len(suffix) || !strings.HasSuffix(slice, suffix) {
		return "", fmt.Errorf("invalid slice name: %s", slice)
	}

	// Path-separators are not allowed.
	if strings.Contains(slice, "/") {
		return "", fmt.Errorf("invalid slice name: %s", slice)
	}

	var path, prefix string
	sliceName := strings.TrimSuffix(slice, suffix)
	// if input was -.slice, we should just return root now
	if sliceName == "-" {
		return "/", nil
	}
	for _, component := range strings.Split(sliceName, "-") {
		// test--a.slice isn't permitted, nor is -test.slice.
		if component == "" {
			return "", fmt.Errorf("invalid slice name: %s", slice)
		}

		// Append the component to the path and to the prefix.
		path += "/" + prefix + component + suffix
		prefix += component + "-"
	}
	return path, nil
}
