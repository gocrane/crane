package nodelocal

import (
	"errors"
	"fmt"
	"math"
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
	registerCollector(cpuCollectorName, []types.MetricName{types.MetricNameCpuTotalUsage, types.MetricNameCpuTotalUtilization}, collectCPU)
	registerCollector(cpuLoadCollectorName, []types.MetricName{types.MetricNameCpuLoad1Min, types.MetricNameCpuLoad5Min, types.MetricNameCpuLoad15Min}, collectCPULoad)
}

type CpuTimeStampState struct {
	stat      cpu.TimesStat
	timestamp time.Time
}

func collectCPU(nodeState *nodeState) (map[string][]common.TimeSeries, error) {
	var now = time.Now()

	// if the cpu core number is not set, to initialize it
	if nodeState.cpuCoreNumbers == 0 {
		if cpuInfos, err := cpu.Info(); err != nil {
			return nil, err
		} else {
			nodeState.cpuCoreNumbers = uint64(len(cpuInfos))
		}
	}

	stats, err := cpu.Times(false)
	if err != nil {
		return nil, err
	}

	if len(stats) != 1 {
		return nil, fmt.Errorf("len stat is not 1")
	}

	currentCpuState := &CpuTimeStampState{
		stat:      stats[0],
		timestamp: now,
	}

	// if the latest cpu state is empty, to initialize it
	if nodeState.latestCpuState == nil {
		nodeState.latestCpuState = currentCpuState
		return nil, errors.New(types.CollectInitErrorText)
	}

	usagePercent := calculateBusy(nodeState.latestCpuState.stat, currentCpuState.stat)
	usageCore := usagePercent * float64(nodeState.cpuCoreNumbers) * 1000 / types.MaxPercentage

	klog.V(6).Infof("CpuCollector collected,usagePercent %v, usageCore %v", usagePercent, usageCore)

	nodeState.latestCpuState = currentCpuState

	var data = make(map[string][]common.TimeSeries, 2)
	data[string(types.MetricNameCpuTotalUsage)] = []common.TimeSeries{{Samples: []common.Sample{{Value: usageCore, Timestamp: now.Unix()}}}}
	data[string(types.MetricNameCpuTotalUtilization)] = []common.TimeSeries{{Samples: []common.Sample{{Value: usagePercent, Timestamp: now.Unix()}}}}

	return data, nil
}

func collectCPULoad(_ *nodeState) (map[string][]common.TimeSeries, error) {
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

func calculateBusy(stat1 cpu.TimesStat, stat2 cpu.TimesStat) float64 {
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
