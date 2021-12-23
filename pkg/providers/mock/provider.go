package mock

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/log"
	"github.com/gocrane/crane/pkg/providers"
)

var logger = log.Logger()

var _ providers.Interface = &inMemory{}

type inMemory struct {
	samples      []common.Sample
	currentIndex int
}

// NewProvider returns a new mock provider
func NewProvider(config *providers.MockConfig) (providers.Interface, error) {
	if config == nil {
		return nil, fmt.Errorf("nil mock config")
	}
	r, err := os.Open(config.SeedFile)
	if err != nil {
		logger.Error(err, "Failed to open seed file", "seedFile", config.SeedFile)
		return nil, err
	}
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		logger.Error(err, "Failed to read seed file", "seedFile", config.SeedFile)
		return nil, err
	}
	reader := csv.NewReader(bytes.NewBuffer(buf))
	records, _ := reader.ReadAll()
	records = records[1:]

	im := &inMemory{
		samples: []common.Sample{},
	}

	now := time.Now().Truncate(time.Minute)
	var seconds int64
	for i := len(records) / 2; i < len(records); i++ {
		timestamp, _ := strconv.ParseInt(records[i][0], 10, 64)
		t := time.Unix(timestamp, 0)
		if now.Hour() == t.Hour() && now.Minute() == t.Minute() {
			im.currentIndex = i
			seconds = now.Unix() - t.Unix()
			break
		}
	}

	for i := 0; i < len(records); i++ {
		timestamp, _ := strconv.ParseInt(records[i][0], 10, 64)
		timestamp += seconds
		s := common.Sample{}
		s.Timestamp = timestamp
		s.Value, _ = strconv.ParseFloat(records[i][1], 64)
		im.samples = append(im.samples, s)
	}

	return im, nil
}

// GetTimeSeries GetTimeSeries
func (im inMemory) GetTimeSeries(_ string, _ []common.QueryCondition, start time.Time, end time.Time, step time.Duration) ([]*common.TimeSeries, error) {
	var next time.Time
	var samples []common.Sample
	var i int

	logger.Info("GetTimeSeries from imMemory provider", "range", fmt.Sprintf(" [%d, %d]", start.Unix(), end.Unix()))

	for i = range im.samples {
		t := time.Unix(im.samples[i].Timestamp, 0)
		if !t.Before(start) && !t.After(end) {
			samples = append(samples, im.samples[i])
			next = t.Add(step)
			i++
			break
		}
	}

	for ; i < len(im.samples); i++ {
		t := time.Unix(im.samples[i].Timestamp, 0)
		if !t.After(end) && !t.Before(next) {
			if t == next {
				samples = append(samples, im.samples[i])
			}
			next = t.Add(step)
		} else if t.After(end) {
			break
		}
	}

	return []*common.TimeSeries{{Labels: []common.Label{}, Samples: samples}}, nil
}

func (im inMemory) GetLatestTimeSeries(_ string, _ []common.QueryCondition) ([]*common.TimeSeries, error) {
	i := im.currentIndex
	now := time.Now().Unix()
	for ; i < len(im.samples); i++ {
		if im.samples[i].Timestamp > now {
			break
		}
	}
	return []*common.TimeSeries{{
		Samples: im.samples[i-1 : i],
	}}, nil
}

// QueryTimeSeries QueryTimeSeries
func (im inMemory) QueryTimeSeries(_ string, start time.Time, end time.Time, step time.Duration) ([]*common.TimeSeries, error) {
	return im.GetTimeSeries("", nil, start, end, step)
}

// QueryLatestTimeSeries QueryLatestTimeSeries
func (im inMemory) QueryLatestTimeSeries(_ string) ([]*common.TimeSeries, error) {
	return im.GetLatestTimeSeries("", nil)
}
