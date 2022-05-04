package internal

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vpa_types "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"

	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/metricquery"
)

const (
	SupportedCheckpointVersion = "v1"
)

// todo: later to remove to api
type MetricNamerModelCheckpoint struct {
	Metric *metricquery.Metric
	// Last update time of the checkpoint data
	LastUpdateTime metav1.Time
	// FirstSampleStart of the model
	FirstSampleStart metav1.Time
	// LastSampleStart of the model
	LastSampleStart metav1.Time
	// SampleInterval of the model
	SampleInterval metav1.Duration
	// TotalSamplesCount of the model
	TotalSamplesCount uint64
	// HistogramModel is the percentile histogram model, only support percentile algorithm model now
	HistogramModel *vpa_types.HistogramCheckpoint
	// Version is the checkpoint version, different versions maybe have different formats.
	Version string
}

type CheckpointContext struct {
	Namer metricnaming.MetricNamer
	Data  *MetricNamerModelCheckpoint
}
