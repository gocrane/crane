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
	registerCollector(memoryCollectorName, []types.MetricName{types.MetricNameMemoryTotalUsage, types.MetricNameMemoryTotalUtilization}, collectMemory)
}

func collectMemory(_ *nodeLocalContext) (map[string][]common.TimeSeries, error) {
	var now = time.Now()
	stat, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	if stat == nil {
		return nil, fmt.Errorf("memory stat is nil")
	}

	usage := stat.Total - stat.Available
	usagePercent := float64(usage) / float64(stat.Total) * types.MaxPercentage

	klog.V(6).Infof("MemoryCollector collected, total %d, Free %d, Available %d, usagePercent %.2f, usageCore %d",
		stat.Total, stat.Free, stat.Available, usagePercent, usage)

	var data = make(map[string][]common.TimeSeries, 2)
	data[string(types.MetricNameMemoryTotalUsage)] = []common.TimeSeries{{Samples: []common.Sample{{Value: float64(usage), Timestamp: now.Unix()}}}}
	data[string(types.MetricNameMemoryTotalUtilization)] = []common.TimeSeries{{Samples: []common.Sample{{Value: usagePercent, Timestamp: now.Unix()}}}}

	return data, nil
}
