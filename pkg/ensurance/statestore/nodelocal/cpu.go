package nodelocal

import (
	"fmt"
	"math"
	"time"

	"github.com/shirou/gopsutil/cpu"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/statestore/types"
)

const (
	cpuCollectorName = "cpu"
	MAX_PERCENTAGE   = 100
	MIN_PERCENTAGE   = 0
)

func init() {
	registerMetrics(cpuCollectorName, []types.MetricName{types.MetricNameCpuTotalUsage, types.MetricNameCpuTotalUtilization}, NewCPUCollector)
}

type CpuTimeStampState struct {
	stat      cpu.TimesStat
	timestamp time.Time
}

type CpuCollector struct {
	cpuState       *CpuTimeStampState
	cpuCoreNumbers uint64
	data           map[string][]common.TimeSeries
}

// NewCPUCollector returns a new Collector exposing kernel/system statistics.
func NewCPUCollector(_ corelisters.PodLister) (nodeLocalCollector, error) {
	var cpuCoreNumbers uint64
	if cpuInfos, err := cpu.Info(); err != nil {
		return nil, err
	} else {
		cpuCoreNumbers = uint64(len(cpuInfos))
	}

	klog.V(2).Infof("NewCPUCollector, cpuCoreNumbers: %v", cpuCoreNumbers)

	var data = make(map[string][]common.TimeSeries)

	return &CpuCollector{cpuCoreNumbers: cpuCoreNumbers, data: data}, nil
}

func (c *CpuCollector) collect() (map[string][]common.TimeSeries, error) {
	var now = time.Now()
	stats, err := cpu.Times(false)
	if err != nil {
		return map[string][]common.TimeSeries{}, err
	}

	if len(stats) != 1 {
		return map[string][]common.TimeSeries{}, fmt.Errorf("len stat is not 1")
	}

	nowCpuState := &CpuTimeStampState{
		stat:      stats[0],
		timestamp: now,
	}

	if c.cpuState == nil {
		c.cpuState = nowCpuState
		return map[string][]common.TimeSeries{}, fmt.Errorf("collect_init")
	}

	usagePercent := calculateBusy(c.cpuState.stat, nowCpuState.stat)
	usageCore := usagePercent * float64(c.cpuCoreNumbers) * 1000 / 100

	klog.V(6).Infof("CpuCollector collected,usagePercent %v, usageCore %v", usagePercent, usageCore)

	c.cpuState = nowCpuState

	c.data[string(types.MetricNameCpuTotalUsage)] = []common.TimeSeries{{Samples: []common.Sample{{Value: usageCore, Timestamp: now.Unix()}}}}
	c.data[string(types.MetricNameCpuTotalUtilization)] = []common.TimeSeries{{Samples: []common.Sample{{Value: usagePercent, Timestamp: now.Unix()}}}}
	return c.data, nil
}

func (c *CpuCollector) name() string {
	return cpuCollectorName
}

func calculateBusy(stat1 cpu.TimesStat, stat2 cpu.TimesStat) float64 {
	stat1All, stat1Busy := getAllBusy(stat1)
	stat2All, stat2Busy := getAllBusy(stat2)

	if stat2Busy <= stat1Busy {
		return MIN_PERCENTAGE
	}
	if stat2All <= stat1All {
		return MAX_PERCENTAGE
	}
	return math.Min(MAX_PERCENTAGE, math.Max(MIN_PERCENTAGE, (stat2Busy-stat1Busy)/(stat2All-stat1All)*100))
}

func getAllBusy(stat cpu.TimesStat) (float64, float64) {
	busy := stat.User + stat.System + stat.Nice + stat.Iowait + stat.Irq + stat.Softirq + stat.Steal
	return busy + stat.Idle, busy
}
