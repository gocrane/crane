package executor

type ReleaseResource map[WatermarkMetric]float64

func (r ReleaseResource) Add(new ReleaseResource) {
	for metric, value := range new {
		if _, ok := r[metric]; !ok {
			r[metric] = 0.0
		}
		r[metric] += value
	}
}
