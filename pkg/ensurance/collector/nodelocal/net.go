package nodelocal

import (
	"time"

	"github.com/shirou/gopsutil/net"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/collector/types"
)

const (
	netioCollectorName = "netio"
)

func init() {
	registerCollector(netioCollectorName, []types.MetricName{types.MetricNetworkReceiveKiBPS, types.MetricNetworkSentKiBPS, types.MetricNetworkReceivePckPS, types.MetricNetworkSentPckPS, types.MetricNetworkDropIn, types.MetricNetworkDropOut}, collectNetIO)
}

type NetTimeStampState struct {
	stat      net.IOCountersStat
	timestamp time.Time
}

// NetInterfaceUsage records the network usage
type NetInterfaceUsage struct {
	// ReceiveKibps is the kilobits per second for ingress
	ReceiveKibps float64
	// SentKibps is the kilobits per second for egress
	SentKibps float64
	// ReceivePckps is the package per second for ingress
	ReceivePckps float64
	// SentPckps is the package per second for egress
	SentPckps float64
	// DropIn is the package dropped per second for ingress
	DropIn float64
	// DropOut is the package dropped per second for egress
	DropOut float64
}

func collectNetIO(nodeLocalContext *nodeLocalContext) (map[string][]common.TimeSeries, error) {
	var now = time.Now()
	nodeState := nodeLocalContext.nodeState

	netIOStats, err := net.IOCounters(true)
	if err != nil {
		klog.Errorf("Failed to collect net io resource: %v", err)
		return nil, err
	}

	var netReceiveKiBpsTimeSeries []common.TimeSeries
	var netSentKiBpsTimeSeries []common.TimeSeries
	var netReceivePckpsTimeSeries []common.TimeSeries
	var netSentPckpsTimeSeries []common.TimeSeries
	var netDropInTimeSeries []common.TimeSeries
	var netDropOutTimeSeries []common.TimeSeries

	var currentNetStates = make(map[string]NetTimeStampState)
	for _, v := range netIOStats {
		if v.Name == "" {
			continue
		}

		if !nodeState.ifaces.Has(v.Name) {
			continue
		}

		currentNetStates[v.Name] = NetTimeStampState{stat: v, timestamp: now}
		if vv, ok := nodeState.latestNetStates[v.Name]; ok {
			netIOUsage := calculateNetIO(vv, currentNetStates[v.Name])
			netReceiveKiBpsTimeSeries = append(netReceiveKiBpsTimeSeries, common.TimeSeries{Labels: []common.Label{{Name: "NetInterface", Value: v.Name}}, Samples: []common.Sample{{Value: netIOUsage.ReceiveKibps, Timestamp: now.Unix()}}})
			netSentKiBpsTimeSeries = append(netSentKiBpsTimeSeries, common.TimeSeries{Labels: []common.Label{{Name: "NetInterface", Value: v.Name}}, Samples: []common.Sample{{Value: netIOUsage.SentKibps, Timestamp: now.Unix()}}})
			netReceivePckpsTimeSeries = append(netReceivePckpsTimeSeries, common.TimeSeries{Labels: []common.Label{{Name: "NetInterface", Value: v.Name}}, Samples: []common.Sample{{Value: netIOUsage.ReceivePckps, Timestamp: now.Unix()}}})
			netSentPckpsTimeSeries = append(netSentPckpsTimeSeries, common.TimeSeries{Labels: []common.Label{{Name: "NetInterface", Value: v.Name}}, Samples: []common.Sample{{Value: netIOUsage.SentPckps, Timestamp: now.Unix()}}})
			netDropInTimeSeries = append(netDropInTimeSeries, common.TimeSeries{Labels: []common.Label{{Name: "NetInterface", Value: v.Name}}, Samples: []common.Sample{{Value: netIOUsage.DropIn, Timestamp: now.Unix()}}})
			netDropOutTimeSeries = append(netDropOutTimeSeries, common.TimeSeries{Labels: []common.Label{{Name: "NetInterface", Value: v.Name}}, Samples: []common.Sample{{Value: netIOUsage.DropOut, Timestamp: now.Unix()}}})
		}
	}

	nodeState.latestNetStates = currentNetStates

	var data = make(map[string][]common.TimeSeries, 6)
	data[string(types.MetricNetworkReceiveKiBPS)] = netReceiveKiBpsTimeSeries
	data[string(types.MetricNetworkSentKiBPS)] = netSentKiBpsTimeSeries
	data[string(types.MetricNetworkReceivePckPS)] = netReceivePckpsTimeSeries
	data[string(types.MetricNetworkSentPckPS)] = netSentPckpsTimeSeries
	data[string(types.MetricNetworkDropIn)] = netDropInTimeSeries
	data[string(types.MetricNetworkDropOut)] = netDropOutTimeSeries

	return data, nil
}

// calculateNetIO calculate net io usage
func calculateNetIO(stat1 NetTimeStampState, stat2 NetTimeStampState) NetInterfaceUsage {

	duration := float64(stat2.timestamp.Unix() - stat1.timestamp.Unix())

	return NetInterfaceUsage{
		ReceiveKibps: float64(stat2.stat.BytesRecv-stat1.stat.BytesRecv) * 8 / 1000 / duration,
		SentKibps:    float64(stat2.stat.BytesSent-stat1.stat.BytesSent) * 8 / 1000 / duration,
		ReceivePckps: float64(stat2.stat.PacketsRecv-stat1.stat.PacketsRecv) / duration,
		SentPckps:    float64(stat2.stat.PacketsSent-stat1.stat.PacketsSent) / duration,
		DropIn:       float64(stat2.stat.Dropin-stat1.stat.Dropin) / duration,
		DropOut:      float64(stat2.stat.Dropout-stat1.stat.Dropout) / duration,
	}
}
