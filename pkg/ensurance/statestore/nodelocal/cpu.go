package nodelocal

import (
	"fmt"
	"math"
	"time"

	"github.com/shirou/gopsutil/cpu"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/statestore/types"
	"github.com/gocrane/crane/pkg/utils/log"
)

const (
	cpuCollectorName = "cpu"
	MAX_PERCENTAGE   = 100
	MIN_PERCENTAGE   = 0
)

func init() {
	registerMetrics(cpuCollectorName, []types.MetricName{types.MetricNamCpuTotalUsage, types.MetricNamCpuTotalUtilization}, NewCPUCollector)
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
func NewCPUCollector() (nodeLocalCollector, error) {
	var cpuCoreNumbers uint64
	if cpuInfos, err := cpu.Info(); err != nil {
		return nil, err
	} else {
		cpuCoreNumbers = uint64(len(cpuInfos))
	}

	log.Logger().V(2).Info("NewCPUCollector", "cpuCoreNumbers", cpuCoreNumbers)

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
	log.Logger().V(4).Info("CpuCollector collect", "usagePercent", usagePercent)
	usageCore := usagePercent * float64(c.cpuCoreNumbers) * 1000 / 100

	c.cpuState = nowCpuState

	c.data[string(types.MetricNamCpuTotalUsage)] = []common.TimeSeries{{Samples: []common.Sample{{Value: usageCore, Timestamp: now.Unix()}}}}
	c.data[string(types.MetricNamCpuTotalUtilization)] = []common.TimeSeries{{Samples: []common.Sample{{Value: usagePercent, Timestamp: now.Unix()}}}}
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
