package metrics

import (
	"github.com/gocrane/crane/pkg/utils"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
	"sync"
	"sync/atomic"
	"time"
)

var (
	nodeExtCPUUsageDesc = prometheus.NewDesc("node_ext_cpu_usage_seconds_total",
		"Cumulative cpu time consumed by the node in core-seconds",
		[]string{"node"},
		nil)
	nodeCPUCanBeReusedDesc = prometheus.NewDesc("node_cpu_can_be_reused_seconds",
		"Cumulative cpu time consumed by the node in core-seconds",
		[]string{"node"},
		nil)
)

var (
	cpuUsageCounter uint64 = 0
)

var registerResourceMetricsOnce sync.Once

func RegisterResourceMetrics(nodeName string, cpuStateProvider *utils.CpuStateProvider) {
	registerResourceMetricsOnce.Do(func() {
		legacyregistry.RawMustRegister(NewResourceMetricsCollector(nodeName, cpuStateProvider))
	})
}

// NewResourceMetricsCollector returns a metrics.StableCollector which exports resource metrics
func NewResourceMetricsCollector(nodeName string, cpuStateProvider *utils.CpuStateProvider) prometheus.Collector {
	return &resourceMetricsCollector{
		node:        nodeName,
		cpuStateProvider: cpuStateProvider,
	}
}

type resourceMetricsCollector struct {
	node        string
	cpuStateProvider *utils.CpuStateProvider
}

// DescribeWithStability implements metrics.StableCollector
func (rc *resourceMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	//ch <- containerExtCPUUsageDesc
	ch <- nodeExtCPUUsageDesc
	ch <- nodeCPUCanBeReusedDesc
}

// CollectWithStability implements metrics.StableCollector
// Since new containers are frequently created and removed, using the Gauge would
// leak metric collectors for containers or pods that no longer exist.  Instead, implement
// custom collector in a way that only collects metrics for active containers.
func (rc *resourceMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	var cpuIdleCanBeReused float64 = 0
	var offlineCpuUsageIncrease uint64 = 0
	var offlineCpuUsageAvg float64 = 0
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		cpuIdleCanBeReused = rc.cpuStateProvider.CollectCpuCoresCanBeReused()
	}()
	go func() {
		defer wg.Done()
		offlineCpuUsageIncrease, offlineCpuUsageAvg = rc.cpuStateProvider.GetExtCpuUsage()
	}()
	lastTime := time.Now()
	wg.Wait()

	atomic.AddUint64(&cpuUsageCounter, offlineCpuUsageIncrease)
	rc.collectNodeExtCPUMetrics(ch, &lastTime)
	rc.collectCpuCoresCanBeReusedMetrics(ch, cpuIdleCanBeReused + offlineCpuUsageAvg)
}

func (rc *resourceMetricsCollector) collectCpuCoresCanBeReusedMetrics(ch chan<- prometheus.Metric, value float64) {
	ch <- metrics.NewLazyMetricWithTimestamp(time.Now(),
		prometheus.MustNewConstMetric(nodeCPUCanBeReusedDesc, prometheus.GaugeValue, value, rc.node))
}

func (rc *resourceMetricsCollector) collectNodeExtCPUMetrics(ch chan<- prometheus.Metric, t *time.Time) {
	ch <- metrics.NewLazyMetricWithTimestamp(*t,
		prometheus.MustNewConstMetric(nodeExtCPUUsageDesc, prometheus.CounterValue,
			float64(atomic.LoadUint64(&cpuUsageCounter))/float64(time.Second), rc.node))
}
