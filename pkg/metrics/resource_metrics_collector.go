package metrics

import (
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	cadvisorapiv2 "github.com/google/cadvisor/info/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/cpu"
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"

	"github.com/gocrane/crane/pkg/ensurance/collector/cadvisor"
	"github.com/gocrane/crane/pkg/ensurance/collector/nodelocal"
	"github.com/gocrane/crane/pkg/ensurance/collector/types"
	"github.com/gocrane/crane/pkg/utils"
)

var (
	nodeExtCPUUsageDesc = prometheus.NewDesc("node_ext_cpu_usage_seconds_total",
		"The cpu seconds that used by ext-containers",
		[]string{"node"},
		nil)
	nodeCPUCannotBeReclaimedDesc = prometheus.NewDesc("node_cpu_cannot_be_reclaimed_seconds",
		"The cpu seconds that cannot be reclaimed",
		[]string{"node"},
		nil)
)

var (
	cpuUsageCounter uint64 = 0
)

var registerResourceMetricsOnce sync.Once

func RegisterResourceMetrics(nodeName string, podLister corelisters.PodLister, manager cadvisor.Manager, exclusiveCPUSet func() cpuset.CPUSet) {
	registerResourceMetricsOnce.Do(func() {
		legacyregistry.RawMustRegister(NewResourceMetricsCollector(nodeName, podLister, manager, exclusiveCPUSet))
	})
}

// NewResourceMetricsCollector returns a metrics.StableCollector which exports resource metrics
func NewResourceMetricsCollector(nodeName string, podLister corelisters.PodLister, manager cadvisor.Manager, exclusiveCPUSet func() cpuset.CPUSet) prometheus.Collector {
	return &resourceMetricsCollector{
		node:            nodeName,
		manager:         manager,
		podLister:       podLister,
		exclusiveCPUSet: exclusiveCPUSet,
	}
}

type resourceMetricsCollector struct {
	node           string
	manager        cadvisor.Manager
	podLister      corelisters.PodLister
	cpuCoreNumbers uint64

	latestContainersStates map[string]cadvisor.ContainerState
	latestNodeStates       *nodelocal.CpuTimeStampState
	exclusiveCPUSet        func() cpuset.CPUSet
}

func (rc *resourceMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- nodeExtCPUUsageDesc
	ch <- nodeCPUCannotBeReclaimedDesc
}

func (rc *resourceMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	var cpuCannotBeReclaimed float64 = 0
	var offlineCpuUsageIncrease uint64 = 0
	var offlineCpuUsageAvg float64 = 0
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		cpuCannotBeReclaimed = rc.getNodeCannotBeReclaimedCpu()
	}()
	go func() {
		defer wg.Done()
		offlineCpuUsageIncrease, offlineCpuUsageAvg = rc.getExtCpuUsage()
	}()
	lastTime := time.Now()
	wg.Wait()

	atomic.AddUint64(&cpuUsageCounter, offlineCpuUsageIncrease)
	rc.collectNodeExtCPUMetrics(ch, &lastTime)
	rc.collectCpuCoresCanBeReusedMetrics(ch, cpuCannotBeReclaimed+offlineCpuUsageAvg)
}

func (rc *resourceMetricsCollector) collectCpuCoresCanBeReusedMetrics(ch chan<- prometheus.Metric, value float64) {
	ch <- metrics.NewLazyMetricWithTimestamp(time.Now(),
		prometheus.MustNewConstMetric(nodeCPUCannotBeReclaimedDesc, prometheus.GaugeValue, value, rc.node))
}

func (rc *resourceMetricsCollector) collectNodeExtCPUMetrics(ch chan<- prometheus.Metric, t *time.Time) {
	ch <- metrics.NewLazyMetricWithTimestamp(*t,
		prometheus.MustNewConstMetric(nodeExtCPUUsageDesc, prometheus.CounterValue,
			float64(atomic.LoadUint64(&cpuUsageCounter))/float64(time.Second), rc.node))
}

func (rc *resourceMetricsCollector) getExtCpuUsage() (uint64, float64) {
	var containerStates = make(map[string]cadvisor.ContainerState)

	allPods, err := rc.podLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list all pods: %v", err)
		return 0, 0
	}
	var extResCpuUseIncrease uint64 = 0
	var extResCpuUseSample float64 = 0

	for _, pod := range allPods {
		var now = time.Now()
		containers, err := rc.manager.GetContainerInfoV2(types.GetCgroupPath(pod), cadvisorapiv2.RequestOptions{
			IdType:    cadvisorapiv2.TypeName,
			Count:     1,
			Recursive: true,
		})
		if err != nil {
			klog.Errorf("GetContainerInfoV2 failed: %v", err)
			continue
		}
		for key, v := range containers {
			containerId := utils.GetContainerIdFromKey(key)
			containerName := cadvisor.GetContainerNameFromPod(pod, containerId)
			klog.V(6).Infof("containerId: %s, containerName: %s", containerId, containerName)
			// Filter the sandbox container
			if (containerId != "") && (containerName == "") {
				continue
			}

			_, hasExtRes := cadvisor.GetContainerExtCpuResFromPod(pod, containerName)
			if !hasExtRes {
				continue
			}

			if state, ok := rc.latestContainersStates[key]; ok {
				cpuUsageIncrease, cpuUsageSample := caculateCPUUsage(&v, &state)
				extResCpuUseIncrease += cpuUsageIncrease
				extResCpuUseSample += cpuUsageSample
			}
			containerStates[key] = cadvisor.ContainerState{Stat: v, Timestamp: now}
		}
	}
	rc.latestContainersStates = containerStates

	return extResCpuUseIncrease, extResCpuUseSample
}

func (rc *resourceMetricsCollector) getNodeCannotBeReclaimedCpu() float64 {
	var now = time.Now()
	// if the cpu core number is not set, to initialize it
	if rc.cpuCoreNumbers == 0 {
		if cpuInfos, err := cpu.Info(); err != nil {
			return 0
		} else {
			rc.cpuCoreNumbers = uint64(len(cpuInfos))
		}
	}

	totalCpuStat, err := cpu.Times(false)
	if err != nil {
		klog.Errorf("Failed to get cpu totalCpuStat: %v", err)
		return 0
	}
	if len(totalCpuStat) != 1 {
		return 0
	}

	perCpuStats, err := cpu.Times(true)
	if err != nil {
		klog.Errorf("Failed to get cpu perCpuStats: %v", err)
	}

	if len(perCpuStats) == 0 {
		return 0
	}

	currentCpuState := &nodelocal.CpuTimeStampState{
		Timestamp: now,
		Stat:      totalCpuStat[0],
	}

	statsMap := make(map[int]cpu.TimesStat)
	for _, stat := range perCpuStats {
		cpuId, err := strconv.ParseInt(strings.TrimPrefix(stat.CPU, "cpu"), 10, 64)
		if err != nil {
			continue
		}
		statsMap[int(cpuId)] = stat
	}
	currentCpuState.PerStat = statsMap

	// if the latest cpu state is empty, to initialize it
	if rc.latestNodeStates == nil {
		rc.latestNodeStates = currentCpuState
		return 0
	}

	usagePercent := nodelocal.CalculateBusy(rc.latestNodeStates.Stat, currentCpuState.Stat)
	usageCore := usagePercent * float64(rc.cpuCoreNumbers) * 1000 / types.MaxPercentage

	cpuSet := rc.exclusiveCPUSet()
	var exclusiveCPUIdle float64 = 0

	for cpuId, stat := range currentCpuState.PerStat {
		if !cpuSet.Contains(cpuId) {
			continue
		}
		if oldStat, ok := rc.latestNodeStates.PerStat[cpuId]; ok {
			exclusiveCPUIdle += nodelocal.CalculateIdle(oldStat, stat) * 1000 / types.MaxPercentage
		}
	}

	klog.V(6).Infof("CpuCollector collected,usagePercent %v, usageCore %v, exclusiveCPUIdle %v", usagePercent, usageCore, exclusiveCPUIdle)
	rc.latestNodeStates = currentCpuState

	return usageCore + exclusiveCPUIdle
}

func caculateCPUUsage(info *cadvisorapiv2.ContainerInfo, state *cadvisor.ContainerState) (uint64, float64) {
	if info == nil ||
		state == nil ||
		len(info.Stats) == 0 {
		return 0, 0
	}
	cpuUsageIncrease := info.Stats[0].Cpu.Usage.Total - state.Stat.Stats[0].Cpu.Usage.Total
	timeIncrease := info.Stats[0].Timestamp.UnixNano() - state.Stat.Stats[0].Timestamp.UnixNano()
	cpuUsageSample := float64(cpuUsageIncrease) / float64(timeIncrease)
	return cpuUsageIncrease, cpuUsageSample
}
