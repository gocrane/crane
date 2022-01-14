package nodelocal

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/mem"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/collector/types"
)

const (
	memoryCollectorName = "memory"
)

func init() {
	registerMetrics(memoryCollectorName, []types.MetricName{types.MetricNameMemoryTotalUsage, types.MetricNameMemoryTotalUtilization}, NewMemoryCollector)
}

type MemoryCollector struct {
	data map[string][]common.TimeSeries
}

// NewMemoryCollector returns a new Collector exposing kernel/system statistics.
func NewMemoryCollector(_ *NodeLocalContext) (nodeLocalCollector, error) {

	var data = make(map[string][]common.TimeSeries)

	return &MemoryCollector{data: data}, nil
}

func (c *MemoryCollector) collect() (map[string][]common.TimeSeries, error) {
	var now = time.Now()
	stat, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	if stat == nil {
		return nil, fmt.Errorf("stat is nil")
	}

	usage := stat.Total - stat.Available
	usagePercent := float64(usage) / float64(stat.Total) * 100.0

	klog.V(6).Infof("MemoryCollector collected, total %d, Free %d, Available %d, usagePercent %.2f, usageCore %d",
		stat.Total, stat.Free, stat.Available, usagePercent, usage)

	c.data[string(types.MetricNameMemoryTotalUsage)] = []common.TimeSeries{{Samples: []common.Sample{{Value: float64(usage), Timestamp: now.Unix()}}}}
	c.data[string(types.MetricNameMemoryTotalUtilization)] = []common.TimeSeries{{Samples: []common.Sample{{Value: usagePercent, Timestamp: now.Unix()}}}}

	return c.data, nil
}

func (c *MemoryCollector) name() string {
	return memoryCollectorName
}
