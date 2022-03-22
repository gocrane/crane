package nodelocal

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/load"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/collector/types"
)

const (
	cpuCollectorName     = "cpu"
	cpuLoadCollectorName = "cpuLoad"
)

func init() {
	registerCollector(cpuCollectorName, []types.MetricName{types.MetricNameCpuTotalUsage, types.MetricNameCpuTotalUtilization, types.MetricNameExclusiveCPUIdle}, collectCPU)
	registerCollector(cpuLoadCollectorName, []types.MetricName{types.MetricNameCpuLoad1Min, types.MetricNameCpuLoad5Min, types.MetricNameCpuLoad15Min}, collectCPULoad)
}

type CpuTimeStampState struct {
	Stat      cpu.TimesStat
	PerStat   map[int]cpu.TimesStat
	Timestamp time.Time
}

func collectCPU(nodeLocalContext *nodeLocalContext) (map[string][]common.TimeSeries, error) {
	var now = time.Now()
	nodeState := nodeLocalContext.nodeState
	// if the cpu core number is not set, to initialize it
	if nodeState.cpuCoreNumbers == 0 {
		if cpuInfos, err := cpu.Info(); err != nil {
			return nil, err
		} else {
			nodeState.cpuCoreNumbers = uint64(len(cpuInfos))
		}
	}

	totalCpuStat, err := cpu.Times(false)
	if err != nil {
		return nil, err
	}
	if len(totalCpuStat) != 1 {
		return nil, fmt.Errorf("len totalCpuStat is not 1")
	}

	perCpuStats, err := cpu.Times(true)
	if err != nil {
		return nil, err
	}

	if len(perCpuStats) == 0 {
		return nil, fmt.Errorf("len perCpuStats is 0")
	}

	currentCpuState := &CpuTimeStampState{
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
	if nodeState.latestCpuState == nil {
		nodeState.latestCpuState = currentCpuState
		return nil, errors.New(types.CollectInitErrorText)
	}

	usagePercent := CalculateBusy(nodeState.latestCpuState.Stat, currentCpuState.Stat)
	usageCore := usagePercent * float64(nodeState.cpuCoreNumbers) * 1000 / types.MaxPercentage

	cpuSet := nodeLocalContext.exclusiveCPUSet()
	var exclusiveCPUIdle float64 = 0

	for cpuId, stat := range currentCpuState.PerStat {
		if !cpuSet.Contains(cpuId) {
			continue
		}
		if oldStat, ok := nodeState.latestCpuState.PerStat[cpuId]; ok {
			exclusiveCPUIdle += CalculateIdle(oldStat, stat) * 1000 / types.MaxPercentage
		}
	}

	klog.V(6).Infof("CpuCollector collected,usagePercent %v, usageCore %v, exclusiveCPUIdle %v", usagePercent, usageCore, exclusiveCPUIdle)

	nodeState.latestCpuState = currentCpuState

	var data = make(map[string][]common.TimeSeries, 2)
	data[string(types.MetricNameCpuTotalUsage)] = []common.TimeSeries{{Samples: []common.Sample{{Value: usageCore, Timestamp: now.Unix()}}}}
	data[string(types.MetricNameCpuTotalUtilization)] = []common.TimeSeries{{Samples: []common.Sample{{Value: usagePercent, Timestamp: now.Unix()}}}}
	data[string(types.MetricNameExclusiveCPUIdle)] = []common.TimeSeries{{Samples: []common.Sample{{Value: exclusiveCPUIdle, Timestamp: now.Unix()}}}}

	return data, nil
}

func collectCPULoad(_ *nodeLocalContext) (map[string][]common.TimeSeries, error) {
	var now = time.Now()
	stat, err := load.Avg()
	if err != nil {
		return nil, err
	}

	if stat == nil {
		return nil, fmt.Errorf("cpu load stat is nil")
	}

	klog.V(6).Infof("LoadCollector collected 1minLoad %v, 5minLoad %v, 15minLoad %v", stat.Load1, stat.Load5, stat.Load15)

	var data = make(map[string][]common.TimeSeries, 3)
	data[string(types.MetricNameCpuLoad1Min)] = []common.TimeSeries{{Samples: []common.Sample{{Value: stat.Load1, Timestamp: now.Unix()}}}}
	data[string(types.MetricNameCpuLoad5Min)] = []common.TimeSeries{{Samples: []common.Sample{{Value: stat.Load5, Timestamp: now.Unix()}}}}
	data[string(types.MetricNameCpuLoad15Min)] = []common.TimeSeries{{Samples: []common.Sample{{Value: stat.Load15, Timestamp: now.Unix()}}}}

	return data, nil
}

func CalculateBusy(stat1 cpu.TimesStat, stat2 cpu.TimesStat) float64 {
	stat1All, stat1Busy := getAllBusy(stat1)
	stat2All, stat2Busy := getAllBusy(stat2)

	if stat2Busy <= stat1Busy {
		return types.MinPercentage
	}
	if stat2All <= stat1All {
		return types.MaxPercentage
	}
	return math.Min(types.MaxPercentage, math.Max(types.MinPercentage, (stat2Busy-stat1Busy)/(stat2All-stat1All)*100))
}

func getAllBusy(stat cpu.TimesStat) (float64, float64) {
	busy := stat.User + stat.System + stat.Nice + stat.Iowait + stat.Irq + stat.Softirq + stat.Steal
	return busy + stat.Idle, busy
}

func CalculateIdle(stat1 cpu.TimesStat, stat2 cpu.TimesStat) float64 {
	stat1All, stat1Idle := getAllIdle(stat1)
	stat2All, stat2Idle := getAllIdle(stat2)
	if stat2Idle <= stat1Idle {
		return types.MinPercentage
	}
	if stat2All <= stat1All {
		return types.MaxPercentage
	}

	return math.Min(types.MaxPercentage, math.Max(types.MinPercentage, (stat2Idle-stat1Idle)/(stat2All-stat1All)*100))
}

func getAllIdle(stat cpu.TimesStat) (float64, float64) {
	busy := stat.User + stat.System + stat.Nice + stat.Iowait + stat.Irq + stat.Softirq + stat.Steal
	return busy + stat.Idle, stat.Idle
}
