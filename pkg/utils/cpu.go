package utils

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocrane/crane/pkg/utils/bt"
	cmanager "github.com/google/cadvisor/manager"
	"github.com/shirou/gopsutil/cpu"
	v1 "k8s.io/api/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
	summary "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
)

const (
	MAX_PERCENTAGE    = 100
	MIN_PERCENTAGE    = 0
	ExtResourcePrefix = "ext-resource.node.gocrane.io/%s"
)

type CpuTimeStampState struct {
	stat      map[int]cpu.TimesStat
	timestamp time.Time
}

type BtCpuTimeStampState struct {
	stat      bt.TimesStat
	timestamp time.Time
}

func NewCpuStateProvider(cmanager cmanager.Manager, podLister corelisters.PodLister, useBt bool, exclusiveCPUSet func() cpuset.CPUSet) *CpuStateProvider {
	var cpuCoreNumbers uint64
	if cpuInfos, err := cpu.Info(); err != nil {
		klog.Errorf("Get Cpu info with error: %v", err)
		return nil
	} else {
		cpuCoreNumbers = uint64(len(cpuInfos))
	}
	return &CpuStateProvider{
		podLister:       podLister,
		provider:        NewCadvisorProvider(cmanager, podLister),
		mu:              sync.RWMutex{},
		useBt:           useBt,
		cpuCoreNumbers:  cpuCoreNumbers,
		exclusiveCPUSet: exclusiveCPUSet,
	}
}

type CpuStateProvider struct {
	podLister       corelisters.PodLister
	provider        *CadvisorProvider
	mu              sync.RWMutex
	extPodState     map[summary.PodReference]summary.PodStats
	cpuState        *CpuTimeStampState
	btCpuState      *BtCpuTimeStampState
	exclusiveCPUSet func() cpuset.CPUSet
	useBt           bool
	cpuCoreNumbers  uint64
}

func (rc *CpuStateProvider) GetExtCpuUsage() (uint64, float64) {
	if rc.useBt {
		return rc.getExtCpuUsageFromBt()
	}

	return rc.getExtCpuUsageFromK8s()
}

func (rc *CpuStateProvider) getExtCpuUsageFromBt() (uint64, float64) {
	var now = time.Now()
	stats, err := bt.Times(false)
	if err != nil {
		return 0, 0
	}

	if len(stats) != 1 {
		return 0, 0
	}
	stat := stats[0]
	nowCpuState := &BtCpuTimeStampState{
		stat:      stat,
		timestamp: now,
	}

	if rc.btCpuState == nil {
		rc.btCpuState = nowCpuState
		klog.V(4).Infof("resourceMetricsCollector bt cpu state init")
		return 0, 0
	}
	offlineIncrease, offlinePercent := rc.btCpuState.stat.CalculateBtOffline(stat)
	rc.btCpuState = nowCpuState

	return uint64(offlineIncrease * float64(time.Second)), offlinePercent * float64(rc.cpuCoreNumbers) / 100
}

func (rc *CpuStateProvider) getExtCpuUsageFromK8s() (uint64, float64) {
	var increaseUseTotal uint64 = 0
	var currentUse float64 = 0
	statsSummary, err := rc.provider.GetCPUAndMemoryStats()
	if err != nil {
		klog.ErrorS(err, "Error getting summary for resourceMetric prometheus endpoint")
		return 0, 0
	}
	extraCpu := fmt.Sprintf(ExtResourcePrefix, "cpu")
	var lastTime *time.Time
	for _, pod := range statsSummary.Pods {
		klog.V(5).Infof("Traverside pod %s", pod.PodRef.Name)
		p, err := rc.GetPodByName(pod.PodRef.Namespace, pod.PodRef.Name)
		if err != nil {
			continue
		}
		podHasExtraCpu := false
		var oldPodStats summary.PodStats
		var oldPodStatsExist bool
		func() {
			rc.mu.RLock()
			defer rc.mu.RUnlock()
			oldPodStats, oldPodStatsExist = rc.extPodState[pod.PodRef]
		}()
		for _, container := range pod.Containers {
			if lastTime == nil {
				lastTime = &container.CPU.Time.Time
			}
			if container.CPU.Time.Time.After(*lastTime) {
				lastTime = &container.CPU.Time.Time
			}
			v1container := v1.Container{}
			for _, c := range p.Spec.Containers {
				if c.Name == container.Name {
					v1container = c
					break
				}
			}

			if _, ok := v1container.Resources.Requests[v1.ResourceName(extraCpu)]; ok {
				podHasExtraCpu = true
				klog.V(5).Infof("%s has extra resource", container.Name)
				if oldPodStatsExist {
					for _, oc := range oldPodStats.Containers {
						if oc.Name == container.Name {
							diff := *container.CPU.UsageCoreNanoSeconds - *oc.CPU.UsageCoreNanoSeconds
							increaseUseTotal += diff
							timeIncrease := container.CPU.Time.UnixNano() - oc.CPU.Time.UnixNano()
							currentUse += float64(diff) / float64(timeIncrease)
						}
					}
				}
			}
		}
		if podHasExtraCpu {
			func() {
				rc.mu.Lock()
				defer rc.mu.Unlock()
				rc.extPodState[pod.PodRef] = pod
			}()
		}
	}
	rc.cleanOldExtPodStats(statsSummary.Pods)
	return increaseUseTotal, currentUse
}

func (rc *CpuStateProvider) GetPodByName(namespace, name string) (*v1.Pod, error) {
	return rc.podLister.Pods(namespace).Get(name)
}

func (rc *CpuStateProvider) cleanOldExtPodStats(newPodStats []summary.PodStats) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	for oldPodRef, _ := range rc.extPodState {
		exist := false
		for _, new := range newPodStats {
			if oldPodRef == new.PodRef {
				exist = true
				break
			}
		}
		if !exist {
			delete(rc.extPodState, oldPodRef)
		}
	}
}

func (rc *CpuStateProvider) CollectCpuCoresCanBeReused() float64 {
	var idle float64 = 0
	var now = time.Now()
	stats, err := cpu.Times(true)
	if err != nil {
		return 0
	}

	if len(stats) <= 1 {
		return 0
	}

	statsMap := make(map[int]cpu.TimesStat)

	for _, stat := range stats {
		if stat.CPU == "cpu-total" {
			continue
		}
		cpuId, err := strconv.ParseInt(strings.TrimPrefix(stat.CPU, "cpu"), 10, 64)
		if err != nil {
			continue
		}
		statsMap[int(cpuId)] = stat
	}

	nowCpuState := &CpuTimeStampState{
		stat:      statsMap,
		timestamp: now,
	}

	if rc.cpuState == nil {
		rc.cpuState = nowCpuState
		klog.V(4).Infof("resourceMetricsCollector cpu state init")
		return 0
	}

	cpuSet := rc.exclusiveCPUSet()
	for cpuId, stat := range nowCpuState.stat {
		if cpuSet.Contains(cpuId) {
			continue
		}
		if oldStat, ok := rc.cpuState.stat[cpuId]; ok {
			idle += calculateIdle(oldStat, stat)
		}
	}

	rc.cpuState = nowCpuState

	return idle / 100
}

func calculateIdle(stat1 cpu.TimesStat, stat2 cpu.TimesStat) float64 {
	stat1All, stat1Idle := getAllIdle(stat1)
	stat2All, stat2Idle := getAllIdle(stat2)
	if stat2Idle <= stat1Idle {
		return MIN_PERCENTAGE
	}
	if stat2All <= stat1All {
		return MAX_PERCENTAGE
	}

	return math.Min(MAX_PERCENTAGE, math.Max(MIN_PERCENTAGE, (stat2Idle-stat1Idle)/(stat2All-stat1All)*100))
}

func getAllIdle(stat cpu.TimesStat) (float64, float64) {
	busy := stat.User + stat.System + stat.Nice + stat.Iowait + stat.Irq + stat.Softirq + stat.Steal
	return busy + stat.Idle, stat.Idle
}
