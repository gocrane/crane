package nodelocal

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/load"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/collector/types"
)

const (
	cpuLoadCollectorName = "cpuLoad"
)

func init() {
	registerMetrics(cpuLoadCollectorName, []types.MetricName{types.MetricNameCpuLoad1Min, types.MetricNameCpuLoad5Min, types.MetricNameCpuLoad15Min}, NewLoadCollector)
}

type CpuLoadCollector struct {
	data map[string][]common.TimeSeries
}

// NewLoadCollector returns a new Collector exposing kernel/system statistics.
func NewLoadCollector(_ *NodeLocalContext) (nodeLocalCollector, error) {

	var data = make(map[string][]common.TimeSeries)

	return &CpuLoadCollector{data: data}, nil
}

func (l *CpuLoadCollector) collect() (map[string][]common.TimeSeries, error) {
	var now = time.Now()
	stat, err := load.Avg()
	if err != nil {
		return nil, err
	}

	if stat == nil {
		return nil, fmt.Errorf("stat is nil")
	}

	klog.V(6).Infof("LoadCollector collected 1minLoad %v, 5minLoad %v, 15minLoad %v", stat.Load1, stat.Load5, stat.Load15)

	l.data[string(types.MetricNameCpuLoad1Min)] = []common.TimeSeries{{Samples: []common.Sample{{Value: stat.Load1, Timestamp: now.Unix()}}}}
	l.data[string(types.MetricNameCpuLoad5Min)] = []common.TimeSeries{{Samples: []common.Sample{{Value: stat.Load5, Timestamp: now.Unix()}}}}
	l.data[string(types.MetricNameCpuLoad15Min)] = []common.TimeSeries{{Samples: []common.Sample{{Value: stat.Load15, Timestamp: now.Unix()}}}}

	return l.data, nil
}

func (l *CpuLoadCollector) name() string {
	return cpuLoadCollectorName
}
