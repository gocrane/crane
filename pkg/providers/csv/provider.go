package csv

import (
	"bytes"
	"encoding/csv"
	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/providers"
	"io"
	"io/ioutil"
	"strconv"
	"time"

)

var _ providers.Interface = &file{}

type file struct {
	r  io.Reader
	ts *common.TimeSeries
}

// NewProvider NewProvider
func NewProvider(r io.Reader) (providers.Interface, error) {
	f := &file{r: r}
	if err := f.load(); err != nil {
		return nil, err
	}
	return f, nil
}

// GetTimeSeries GetTimeSeries
func (f *file) GetTimeSeries(_ string, _ []common.QueryCondition, start time.Time, end time.Time, step time.Duration) ([]*common.TimeSeries, error) {
	return []*common.TimeSeries{f.ts}, nil
}

// GetLatestTimeSeries GetLatestTimeSeries
func (f *file) GetLatestTimeSeries(_ string, _ []common.QueryCondition) ([]*common.TimeSeries, error) {
	n := len(f.ts.Samples)
	ts := &common.TimeSeries{
		Labels:  f.ts.Labels,
		Samples: f.ts.Samples[n-1:],
	}
	return []*common.TimeSeries{ts}, nil
}

// QueryTimeSeries QueryTimeSeries
func (f *file) QueryTimeSeries(_ string, _ time.Time, _ time.Time, _ time.Duration) ([]*common.TimeSeries, error) {
	return []*common.TimeSeries{f.ts}, nil
}

// QueryLatestTimeSeries QueryLatestTimeSeries
func (f *file) QueryLatestTimeSeries(_ string) ([]*common.TimeSeries, error) {
	n := len(f.ts.Samples)
	ts := &common.TimeSeries{
		Labels:  f.ts.Labels,
		Samples: f.ts.Samples[n-1:],
	}
	return []*common.TimeSeries{ts}, nil
}

func (f *file) load() error {
	buf, err := ioutil.ReadAll(f.r)
	if err != nil {
		return err
	}

	reader := csv.NewReader(bytes.NewBuffer(buf))
	records, _ := reader.ReadAll()

	var samples []common.Sample
	for i := 1; i < len(records); i++ {
		s := common.Sample{}
		s.Timestamp, _ = strconv.ParseInt(records[i][0], 10, 64)
		s.Value, _ = strconv.ParseFloat(records[i][1], 64)
		samples = append(samples, s)
	}

	f.ts = &common.TimeSeries{
		Labels:  []common.Label{},
		Samples: samples,
	}

	return nil
}
