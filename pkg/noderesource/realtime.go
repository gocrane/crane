package noderesource

import (
	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/utils"
	"github.com/gocrane/crane/pkg/utils/bt"
	cadvisorapiv2 "github.com/google/cadvisor/info/v2"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sync"
	"time"
)

const (
	realtimeCollectorName = "Realtime"
)

func init() {
	klog.Infof("init RealtimeCollector")
	registerMetrics(realtimeCollectorName, NewRealTimeCollection)
}

type CpuTimeStampState struct {
	stat      map[int]cpu.TimesStat
	timestamp time.Time
}

type BtCpuTimeStampState struct {
	stat      bt.TimesStat
	timestamp time.Time
}

type CgroupState struct {
	stat      cadvisorapiv2.ContainerInfo
	timestamp time.Time
}

func NewRealTimeCollection(context *CollectContext) (Collector, error) {
	klog.V(4).Infof("create RealTimeCollector")

	updateCycle := 10 * time.Second

	return &RealTimeCollector{
		stateMap:         make(map[string][]common.TimeSeries),
		updateCycle:      updateCycle,
		cpuStateProvider: context.CpuStateProvider,
	}, nil
}

type RealTimeCollector struct {
	stateMap    map[string][]common.TimeSeries
	updateCycle time.Duration
	sync.RWMutex
	cpuStateProvider *utils.CpuStateProvider
}

func (r *RealTimeCollector) Run(stop <-chan struct{}, stateChan chan struct {
	stateMap      map[string][]MetricTimeSeries
	collectorName string
}) {
	go func() {
		updateTicker := time.NewTicker(r.updateCycle)
		defer updateTicker.Stop()
		for {
			select {
			case <-updateTicker.C:
				stateChan <- struct {
					stateMap      map[string][]MetricTimeSeries
					collectorName string
				}{
					stateMap:      r.collect(),
					collectorName: r.Name(),
				}
			case <-stop:
				klog.Infof("RealTimeCollector exit")
				return
			}
		}
	}()
}

func (r *RealTimeCollector) GetLastState() map[string][]MetricTimeSeries {
	r.RLock()
	defer r.RUnlock()
	result := make(map[string][]MetricTimeSeries)
	for key, timeSeriesList := range r.stateMap {
		result[key] = []MetricTimeSeries{{
			DataSourceName: r.Name(),
			TimeSeriesList: timeSeriesList,
		}}
	}
	return result
}

func (r *RealTimeCollector) collect() map[string][]MetricTimeSeries {
	klog.V(4).Infof("RealTimeCollector start collect")
	result := make(map[string][]MetricTimeSeries)
	r.Lock()
	defer r.Unlock()
	cpuState := r.collectCpuTimeSeries()
	memoryState := r.collectMemoryTimeSeries()
	result[v1.ResourceCPU.String()] = []MetricTimeSeries{{
		DataSourceName: r.Name(),
		TimeSeriesList: cpuState,
	}}
	result[v1.ResourceMemory.String()] = []MetricTimeSeries{{
		DataSourceName: r.Name(),
		TimeSeriesList: memoryState,
	}}
	r.stateMap[v1.ResourceCPU.String()] = cpuState
	r.stateMap[v1.ResourceMemory.String()] = memoryState
	return result
}

func (r *RealTimeCollector) collectMemoryTimeSeries() []common.TimeSeries {
	var now = time.Now()
	stat, err := mem.VirtualMemory()
	if err != nil {
		return nil
	}

	if stat == nil {
		return nil
	}

	usage := stat.Total - stat.Available
	usagePercent := float64(usage) / float64(stat.Total) * 100.0

	klog.V(4).Infof("memory of RealTimeCollector collected, total %d, Free %d, Available %d, usagePercent %.2f, usageCore %d",
		stat.Total, stat.Free, stat.Available, usagePercent, usage)

	return []common.TimeSeries{{Samples: []common.Sample{{Value: float64(usage), Timestamp: now.Unix()}}}}
}

func (r *RealTimeCollector) collectCpuTimeSeries() []common.TimeSeries {
	var cpuIdleCanBeReused float64 = 0
	var offlineCpuUsageIncrease uint64 = 0
	var offlineCpuUsageAvg float64 = 0
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		cpuIdleCanBeReused = r.cpuStateProvider.CollectCpuCoresCanBeReused()
	}()
	go func() {
		defer wg.Done()
		offlineCpuUsageIncrease, offlineCpuUsageAvg = r.cpuStateProvider.GetExtCpuUsage()
	}()
	lastTime := time.Now()
	wg.Wait()

	return []common.TimeSeries{{Samples: []common.Sample{{Value: cpuIdleCanBeReused + offlineCpuUsageAvg, Timestamp: lastTime.Unix()}}}}
}

func (r *RealTimeCollector) Name() string {
	return realtimeCollectorName
}

func GetRealtimeCollectorName() string {
	return realtimeCollectorName
}
