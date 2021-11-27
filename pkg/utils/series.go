package utils

// TimeSeries is a stream of samples that belong to a metric with a set of labels
type TimeSeries struct {
	// A collection of Labels that are attached by monitoring system as metadata
	// for the metrics, which are known as dimensions.
	Labels []Label
	// A collection of Samples in chronological order.
	Samples []Sample
}

// Sample pairs a Value with a Timestamp.
type Sample struct {
	Value     float64
	Timestamp int64
}

// A Label is a Name and Value pair that provides additional information about the metric.
// It is metadata for the metric. For example, Kubernetes pod metrics always have
// 'namespace' label that represents which namespace it belongs to.
type Label struct {
	Name  string
	Value string
}

func Labels2Maps(labels []Label) map[string]string {
	if len(labels) == 0 {
		return make(map[string]string)
	}

	var maps = make(map[string]string)
	for _, v := range labels {
		if v.Name != "" {
			maps[v.Name] = v.Value
		}
	}

	return maps
}
