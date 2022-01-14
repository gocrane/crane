//go:build linux
// +build linux

package nodelocal

import (
	"io/ioutil"
	"time"

	"github.com/shirou/gopsutil/disk"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/collector/types"
)

const (
	diskioCollectorName = "diskio"
	sysBlockPath        = "/sys/block"
)

func init() {
	registerMetrics(diskioCollectorName, []types.MetricName{types.MetricDiskReadKiBPS, types.MetricDiskWriteKiBPS, types.MetricDiskReadIOPS, types.MetricDiskWriteIOPS, types.MetricDiskUtilization}, NewDiskIOCollector)
}

type DiskState struct {
	stat      disk.IOCountersStat
	timestamp time.Time
}

type DiskIOCollector struct {
	diskStates map[string]DiskState
}

type DiskIOUsage struct {
	DiskReadKiBps  float64
	DiskWriteKiBps float64
	DiskReadIOps   float64
	DiskWriteIOps  float64
	Utilization    float64
}

// NewDiskIOCollector returns a new Collector exposing kernel/system statistics.
func NewDiskIOCollector(_ *NodeLocalContext) (nodeLocalCollector, error) {
	return &DiskIOCollector{diskStates: make(map[string]DiskState)}, nil
}

func (d *DiskIOCollector) collect() (map[string][]common.TimeSeries, error) {
	var now = time.Now()

	devices, err := sysBlockDevices(sysBlockPath)
	if err != nil {
		return nil, err
	}

	diskIOStats, err := disk.IOCounters(devices...)
	if err != nil {
		klog.Errorf("Failed to collect disk io resource: %v", err)
		return nil, err
	}

	var diskReadKiBpsTimeSeries []common.TimeSeries
	var diskWriteKiBpsTimeSeries []common.TimeSeries
	var diskReadIOpsTimeSeries []common.TimeSeries
	var diskWriteIOpsTimeSeries []common.TimeSeries
	var diskUtilizationTimeSeries []common.TimeSeries

	var diskStateMap = make(map[string]DiskState)
	for key, v := range diskIOStats {
		diskStateMap[key] = DiskState{stat: v, timestamp: now}
		if vv, ok := d.diskStates[key]; ok {
			diskIOUsage := calculateDiskIO(vv, diskStateMap[key])
			diskReadKiBpsTimeSeries = append(diskReadKiBpsTimeSeries, common.TimeSeries{Labels: []common.Label{{Name: "diskName", Value: key}}, Samples: []common.Sample{{Value: diskIOUsage.DiskReadKiBps, Timestamp: now.Unix()}}})
			diskWriteKiBpsTimeSeries = append(diskWriteKiBpsTimeSeries, common.TimeSeries{Labels: []common.Label{{Name: "diskName", Value: key}}, Samples: []common.Sample{{Value: diskIOUsage.DiskWriteKiBps, Timestamp: now.Unix()}}})
			diskReadIOpsTimeSeries = append(diskReadIOpsTimeSeries, common.TimeSeries{Labels: []common.Label{{Name: "diskName", Value: key}}, Samples: []common.Sample{{Value: diskIOUsage.DiskReadIOps, Timestamp: now.Unix()}}})
			diskWriteIOpsTimeSeries = append(diskWriteIOpsTimeSeries, common.TimeSeries{Labels: []common.Label{{Name: "diskName", Value: key}}, Samples: []common.Sample{{Value: diskIOUsage.DiskWriteIOps, Timestamp: now.Unix()}}})
			diskUtilizationTimeSeries = append(diskUtilizationTimeSeries, common.TimeSeries{Labels: []common.Label{{Name: "diskName", Value: key}}, Samples: []common.Sample{{Value: diskIOUsage.Utilization, Timestamp: now.Unix()}}})
		}
	}

	d.diskStates = diskStateMap

	var storeMap = make(map[string][]common.TimeSeries, 0)
	storeMap[string(types.MetricDiskReadKiBPS)] = diskReadKiBpsTimeSeries
	storeMap[string(types.MetricDiskWriteKiBPS)] = diskWriteKiBpsTimeSeries
	storeMap[string(types.MetricDiskReadIOPS)] = diskReadIOpsTimeSeries
	storeMap[string(types.MetricDiskWriteIOPS)] = diskWriteIOpsTimeSeries
	storeMap[string(types.MetricDiskUtilization)] = diskUtilizationTimeSeries

	return storeMap, nil
}

func (d *DiskIOCollector) name() string {
	return diskioCollectorName
}

// sysBlockDevices lists the device names from /sys/block/<dev>.
func sysBlockDevices(sysBlockPath string) ([]string, error) {
	dirs, err := ioutil.ReadDir(sysBlockPath)
	if err != nil {
		return nil, err
	}
	devices := []string{}
	for _, dir := range dirs {
		devices = append(devices, dir.Name())
	}
	return devices, nil
}

// calculateDiskIO calculate disk io usage
func calculateDiskIO(stat1 DiskState, stat2 DiskState) DiskIOUsage {

	duration := float64(stat2.timestamp.Unix() - stat1.timestamp.Unix())

	return DiskIOUsage{
		DiskReadKiBps:  float64(stat2.stat.ReadBytes-stat1.stat.ReadBytes) / types.UintConversionStep1024 / duration,
		DiskWriteKiBps: float64(stat2.stat.WriteBytes-stat1.stat.WriteBytes) / types.UintConversionStep1024 / duration,
		DiskReadIOps:   float64(stat2.stat.ReadCount-stat1.stat.ReadCount) / duration,
		DiskWriteIOps:  float64(stat2.stat.WriteCount-stat1.stat.WriteCount) / duration,
		Utilization:    (float64(stat2.stat.IoTime-stat1.stat.IoTime) / types.UintConversionStep1000 / duration) * 100,
	}
}
