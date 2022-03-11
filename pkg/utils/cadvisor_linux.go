package utils

import (
	cmemory "github.com/google/cadvisor/cache/memory"
	cadvisorcontainer "github.com/google/cadvisor/container"
	cadvisorapiv2 "github.com/google/cadvisor/info/v2"
	cmanager "github.com/google/cadvisor/manager"
	csysfs "github.com/google/cadvisor/utils/sysfs"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
	statsapi "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	CManager cmanager.Manager
	cManagerOnce sync.Once
)

type CadvisorProvider struct {
	Manager   cmanager.Manager
	podLister corelisters.PodLister
}

func NewCadvisorManager() (cmanager.Manager, error) {
	cManagerOnce.Do(func(){
		klog.Infof("CManager is nil")
		var includedMetrics = cadvisorcontainer.MetricSet{
			cadvisorcontainer.CpuUsageMetrics:         struct{}{},
			cadvisorcontainer.ProcessSchedulerMetrics: struct{}{},
		}

		var allowDynamic bool = true
		var maxHousekeepingInterval time.Duration = 10 * time.Second
		var memCache = cmemory.New(10*time.Minute, nil)
		var sysfs = csysfs.NewRealSysFs()
		var maxHousekeepingConfig = cmanager.HouskeepingConfig{Interval: &maxHousekeepingInterval, AllowDynamic: &allowDynamic}

		m, err := cmanager.New(memCache, sysfs, maxHousekeepingConfig, includedMetrics, http.DefaultClient, []string{CgroupKubePods}, "")
		if err != nil {
			klog.Errorf("Failed to create cadvisor manager start: %v", err.Error())
		}
		if err := m.Start(); err != nil {
			klog.Errorf("Failed to start cadvisor manager: %v", err)
		}
		CManager = m
	})
	return CManager, nil
}

func NewCadvisorProvider(manager cmanager.Manager, podLister corelisters.PodLister) *CadvisorProvider {
	return &CadvisorProvider{
		Manager:   manager,
		podLister: podLister,
	}
}

func (c *CadvisorProvider) GetCPUAndMemoryStats() (*statsapi.Summary, error) {
	var podToStats = map[statsapi.PodReference]*statsapi.PodStats{}

	allPods, err := c.podLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list all pods: %v", err)
		return nil, err
	}

	klog.Infof("allPods len %d", len(allPods))

	for _, pod := range allPods {
		containers, err := c.Manager.GetContainerInfoV2(GetCgroupPath(pod), cadvisorapiv2.RequestOptions{
			IdType:    cadvisorapiv2.TypeName,
			Count:     1,
			Recursive: true,
		})
		if err != nil {
			klog.Errorf("GetContainerInfoV2 failed: %v", err)
			continue
		}

		for key, v := range containers {
			ref := statsapi.PodReference{
				Name:      pod.Name,
				Namespace: pod.Namespace,
				UID:       string(pod.UID),
			}
			// Lookup the PodStats for the pod using the PodRef. If none exists,
			// initialize a new entry.
			podStats, found := podToStats[ref]
			if !found {
				podStats = &statsapi.PodStats{PodRef: ref}
				podToStats[ref] = podStats
			}

			containerId := GetContainerIdFromKey(key)
			containerName := GetContainerNameFromPod(pod, containerId)
			// Filter the sandbox container
			if (containerId != "") && (containerName == "") {
				continue
			}
			podStats.Containers = append(podStats.Containers, *cadvisorInfoToContainerCPUAndMemoryStats(containerName, &v))
		}
	}
	// Add each PodStats to the result.
	result := make([]statsapi.PodStats, 0, len(podToStats))
	for _, podStats := range podToStats {
		result = append(result, *podStats)
	}
	return &statsapi.Summary{
		Pods: result,
	}, nil
}

// cadvisorInfoToContainerCPUAndMemoryStats returns the statsapi.ContainerStats converted
// from the container and filesystem info.
func cadvisorInfoToContainerCPUAndMemoryStats(name string, info *cadvisorapiv2.ContainerInfo) *statsapi.ContainerStats {
	result := &statsapi.ContainerStats{
		StartTime: metav1.NewTime(info.Spec.CreationTime),
		Name:      name,
	}

	cpu, memory := cadvisorInfoToCPUandMemoryStats(info)
	result.CPU = cpu
	result.Memory = memory

	return result
}

func cadvisorInfoToCPUandMemoryStats(info *cadvisorapiv2.ContainerInfo) (*statsapi.CPUStats, *statsapi.MemoryStats) {
	cstat, found := latestContainerStats(info)
	if !found {
		return nil, nil
	}
	var cpuStats *statsapi.CPUStats
	var memoryStats *statsapi.MemoryStats
	cpuStats = &statsapi.CPUStats{
		Time:                 metav1.NewTime(cstat.Timestamp),
		UsageNanoCores:       uint64Ptr(0),
		UsageCoreNanoSeconds: uint64Ptr(0),
	}
	if info.Spec.HasCpu {
		if cstat.CpuInst != nil {
			cpuStats.UsageNanoCores = &cstat.CpuInst.Usage.Total
		}
		if cstat.Cpu != nil {
			cpuStats.UsageCoreNanoSeconds = &cstat.Cpu.Usage.Total
		}
	}
	if info.Spec.HasMemory && cstat.Memory != nil {
		pageFaults := cstat.Memory.ContainerData.Pgfault
		majorPageFaults := cstat.Memory.ContainerData.Pgmajfault
		memoryStats = &statsapi.MemoryStats{
			Time:            metav1.NewTime(cstat.Timestamp),
			UsageBytes:      &cstat.Memory.Usage,
			WorkingSetBytes: &cstat.Memory.WorkingSet,
			RSSBytes:        &cstat.Memory.RSS,
			PageFaults:      &pageFaults,
			MajorPageFaults: &majorPageFaults,
		}
		// availableBytes = memory limit (if known) - workingset
		if !isMemoryUnlimited(info.Spec.Memory.Limit) {
			availableBytes := info.Spec.Memory.Limit - cstat.Memory.WorkingSet
			memoryStats.AvailableBytes = &availableBytes
		}
	} else {
		memoryStats = &statsapi.MemoryStats{
			Time:            metav1.NewTime(cstat.Timestamp),
			WorkingSetBytes: uint64Ptr(0),
		}
	}
	return cpuStats, memoryStats
}

// latestContainerStats returns the latest container stats from cadvisor, or nil if none exist
func latestContainerStats(info *cadvisorapiv2.ContainerInfo) (*cadvisorapiv2.ContainerStats, bool) {
	stats := info.Stats
	if len(stats) < 1 {
		return nil, false
	}
	latest := stats[len(stats)-1]
	if latest == nil {
		return nil, false
	}
	return latest, true
}

func uint64Ptr(i uint64) *uint64 {
	return &i
}

func isMemoryUnlimited(v uint64) bool {
	// Size after which we consider memory to be "unlimited". This is not
	// MaxInt64 due to rounding by the kernel.
	const maxMemorySize = uint64(1 << 62)

	return v > maxMemorySize
}

func GetContainerNameFromPod(pod *v1.Pod, containerId string) string {
	if containerId == "" {
		return ""
	}

	for _, v := range pod.Status.ContainerStatuses {
		strList := strings.Split(v.ContainerID, "//")
		if len(strList) > 0 {
			if strList[len(strList)-1] == containerId {
				return v.Name
			}
		}
	}

	return ""
}

func GetCgroupPath(p *v1.Pod) string {
	var pathArrays = []string{CgroupKubePods}

	switch p.Status.QOSClass {
	case v1.PodQOSGuaranteed:
		pathArrays = append(pathArrays, CgroupPodPrefix+string(p.UID))
	case v1.PodQOSBurstable:
		pathArrays = append(pathArrays, strings.ToLower(string(v1.PodQOSBurstable)), CgroupPodPrefix+string(p.UID))
	case v1.PodQOSBestEffort:
		pathArrays = append(pathArrays, strings.ToLower(string(v1.PodQOSBestEffort)), CgroupPodPrefix+string(p.UID))
	default:
		return ""
	}
	return strings.Join(pathArrays, "/")
}

