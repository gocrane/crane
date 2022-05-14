package metricquery

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
)

type MetricSource string

const (
	PrometheusMetricSource   MetricSource = "prom"
	MetricServerMetricSource MetricSource = "metricserver"
	InfluxDBMetricSource     MetricSource = "influxdb"
)

type MetricType string

const (
	WorkloadMetricType   MetricType = "workload"
	PodMetricType        MetricType = "pod"
	ContainerMetricType  MetricType = "container"
	NodeMetricType       MetricType = "node"
	PromQLMetricType     MetricType = "promql"
	InfluxDBQLMetricType MetricType = "influxdbql"
)

var (
	NotMatchWorkloadError  = fmt.Errorf("metric type %v, but no WorkloadNamerInfo provided", WorkloadMetricType)
	NotMatchContainerError = fmt.Errorf("metric type %v, but no ContainerNamerInfo provided", ContainerMetricType)
	NotMatchPodError       = fmt.Errorf("metric type %v, but no PodNamerInfo provided", PodMetricType)
	NotMatchNodeError      = fmt.Errorf("metric type %v, but no NodeNamerInfo provided", NodeMetricType)
	NotMatchPromError      = fmt.Errorf("metric type %v, but no PromNamerInfo provided", PromQLMetricType)
)

type Metric struct {
	Type MetricType
	// such as cpu/memory, or http_requests
	MetricName string
	// Workload only support for MetricName cpu/memory
	Workload *WorkloadNamerInfo
	// Container only support for MetricName cpu/memory
	Container *ContainerNamerInfo
	// Pod only support for MetricName cpu/memory
	Pod *PodNamerInfo
	// Node only support for MetricName cpu/memory
	Node *NodeNamerInfo
	// Prom can support any MetricName, user give the promQL
	Prom *PromNamerInfo
	// InfluxDB can support any MetricName, user give the influxQL
	InfluxDB *InfluxDBNamerInfo
}

type WorkloadNamerInfo struct {
	Namespace  string
	Kind       string
	Name       string
	APIVersion string
	// used to fetch workload pods and containers, when use metric server, it is required
	Selector labels.Selector
}

type ContainerNamerInfo struct {
	Namespace     string
	WorkloadName  string
	Kind          string
	APIVersion    string
	ContainerName string
	// used to fetch workload pods and containers, when use metric server, it is required
	Selector labels.Selector
}

type PodNamerInfo struct {
	Namespace string
	Name      string
	Selector  labels.Selector
}

type NodeNamerInfo struct {
	Name     string
	Selector labels.Selector
}

type PromNamerInfo struct {
	QueryExpr string
	Namespace string
	Selector  labels.Selector
}

type InfluxDBNamerInfo struct {
	QueryExpr string
	Namespace string
	Selector  labels.Selector
}


func (m *Metric) ValidateMetric() error {
	if m == nil {
		return fmt.Errorf("metric is null")
	}
	switch m.Type {
	case WorkloadMetricType:
		if m.Workload == nil {
			return NotMatchWorkloadError
		}
		if m.Workload.Selector == nil {
			return fmt.Errorf("workload metric type must has selector to select pods")
		}
	case ContainerMetricType:
		if m.Container == nil {
			return NotMatchContainerError
		}
	case PodMetricType:
		if m.Pod == nil {
			return NotMatchPodError
		}
	case NodeMetricType:
		if m.Node == nil {
			return NotMatchNodeError
		}
	case PromQLMetricType:
		if m.Prom == nil {
			return NotMatchPromError
		}
	default:
		return fmt.Errorf("not supported metric type %v, %+v", m.Type, *m)
	}
	return nil
}

func (m *Metric) BuildUniqueKey() string {
	err := m.ValidateMetric()
	if err != nil {
		klog.Errorf("Failed to build unique key, validate metric err: %v", err)
		return ""
	}
	switch m.Type {
	case WorkloadMetricType:
		return m.keyByWorkload()
	case ContainerMetricType:
		return m.keyByContainer()
	case PodMetricType:
		return m.keyByPod()
	case NodeMetricType:
		return m.keyByNode()
	case PromQLMetricType:
		return m.keyByPromQL()
	default:
		klog.Errorf("Failed to build unique key, not supported metric type %v", m.Type)
		return ""
	}
}

func (m *Metric) keyByWorkload() string {
	selectorStr := ""
	if m.Workload.Selector != nil {
		selectorStr = m.Workload.Selector.String()
	}
	return strings.Join([]string{
		string(m.Type),
		strings.ToLower(m.MetricName),
		m.Workload.Kind,
		m.Workload.APIVersion,
		m.Workload.Namespace,
		m.Workload.Name,
		selectorStr}, "_")
}

func (m *Metric) keyByContainer() string {
	selectorStr := ""
	if m.Container.Selector != nil {
		selectorStr = m.Container.Selector.String()
	}
	return strings.Join([]string{
		string(m.Type),
		strings.ToLower(m.MetricName),
		m.Container.Namespace,
		m.Container.WorkloadName,
		m.Container.ContainerName,
		selectorStr}, "_")
}

func (m *Metric) keyByPod() string {
	selectorStr := ""
	if m.Pod.Selector != nil {
		selectorStr = m.Pod.Selector.String()
	}
	return strings.Join([]string{
		string(m.Type),
		strings.ToLower(m.MetricName),
		m.Pod.Namespace,
		m.Pod.Name,
		selectorStr}, "_")
}
func (m *Metric) keyByNode() string {
	selectorStr := ""
	if m.Node.Selector != nil {
		selectorStr = m.Node.Selector.String()
	}
	return strings.Join([]string{
		string(m.Type),
		strings.ToLower(m.MetricName),
		m.Node.Name,
		selectorStr}, "_")
}

func (m *Metric) keyByPromQL() string {
	selectorStr := ""
	if m.Prom.Selector != nil {
		selectorStr = m.Prom.Selector.String()
	}
	return strings.Join([]string{
		string(m.Type),
		m.Prom.Namespace,
		strings.ToLower(m.MetricName),
		m.Prom.QueryExpr,
		selectorStr}, "_")
}

// Query is used to do query for different data source. you can extends it with your data source query
type Query struct {
	Type         MetricSource
	MetricServer *MetricServerQuery
	Prometheus   *PrometheusQuery
	InfluxDB     *InfluxDBQuery
}

// MetricServerQuery is used to do query for metric server
type MetricServerQuery struct {
	Metric *Metric
}

// PrometheusQuery is used to do query for prometheus
type PrometheusQuery struct {
	Query string
}

// InfluxDBQuery is used to do query for prometheus
type InfluxDBQuery struct {
	Query string
}
