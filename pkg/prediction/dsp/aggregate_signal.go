package dsp

import (
	"time"

	"github.com/gocrane/crane/pkg/common"
)

type aggregateSignal struct {
	predictedTimeSeries *common.TimeSeries
	startTime           time.Time
	endTime             time.Time
	lastUpdateTime      time.Time
}

func newAggregateSignal() *aggregateSignal {
	return &aggregateSignal{}
}

func (a *aggregateSignal) setPredictedTimeSeries(ts *common.TimeSeries) {
	n := len(ts.Samples)
	if n > 0 {
		a.startTime = time.Unix(ts.Samples[0].Timestamp, 0)
		a.endTime = time.Unix(ts.Samples[n-1].Timestamp, 0)
		a.predictedTimeSeries = ts
		a.lastUpdateTime = time.Now()
	}
}
