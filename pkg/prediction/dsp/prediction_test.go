package dsp

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/providers/csv"
	"github.com/stretchr/testify/assert"
)

func TestPreProcessTimeSeries(t *testing.T) {
	end := time.Now().Truncate(time.Minute)
	var buf bytes.Buffer
	buf.WriteString("ts,value\n")
	ts := end.Add(-2 * time.Hour)
	buf.WriteString(fmt.Sprintf("%d,0.1\n", ts.Unix()))
	ts = ts.Add(time.Minute)
	buf.WriteString(fmt.Sprintf("%d,0.8\n", ts.Unix()))
	ts = end.Add(-10 * time.Minute)
	buf.WriteString(fmt.Sprintf("%d,1.0\n", ts.Unix()))
	ts = ts.Add(3 * time.Minute)
	buf.WriteString(fmt.Sprintf("%d,1.3\n", ts.Unix()))
	ts = end
	buf.WriteString(fmt.Sprintf("%d,2.0\n", ts.Unix()))
	fmt.Println(buf.String())

	prov, err := csv.NewProvider(strings.NewReader(buf.String()))
	assert.NoError(t, err)

	namer := &metricnaming.GeneralMetricNamer{}
	tsList, err := prov.QueryTimeSeries(namer, end.Add(-3*time.Hour), end, time.Minute)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(tsList))

	timeSeries := tsList[0]
	assert.Equal(t, 5, len(timeSeries.Samples))

	assert.NoError(t, preProcessTimeSeries(timeSeries, &defaultInternalConfig, time.Minute))
	assert.Equal(t, 11, len(timeSeries.Samples))

	for i := 1; i < len(timeSeries.Samples); i++ {
		assert.InEpsilon(t, 0.1, timeSeries.Samples[i].Value-timeSeries.Samples[i-1].Value, 1e-9)
		assert.Equal(t, int64(60), timeSeries.Samples[i].Timestamp-timeSeries.Samples[i-1].Timestamp)
	}

	// Truncate the time series to multiple of 10 minutes.
	assert.NoError(t, preProcessTimeSeries(timeSeries, &defaultInternalConfig, 10*time.Minute))
	assert.Equal(t, 10, len(timeSeries.Samples))

	for i := 1; i < len(timeSeries.Samples); i++ {
		assert.InEpsilon(t, 0.1, timeSeries.Samples[i].Value-timeSeries.Samples[i-1].Value, 1e-9)
		assert.Equal(t, int64(60), timeSeries.Samples[i].Timestamp-timeSeries.Samples[i-1].Timestamp)
	}
}
