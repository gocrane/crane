package dsp

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	epsilon        = 1e-15
	sampleRate     = 1000.0                // Hz
	duration       = 1.0                   // second
	sampleInterval = duration / sampleRate // second
	frequency      = 25.0
	amplitude      = 10.0
	nInputs        = 16
)

var signal *Signal

func init() {
	var x, y []float64
	for i := 0.0; i < duration; i += sampleInterval {
		x = append(x, i)
	}
	for i := range x {
		y = append(y, amplitude*math.Cos(2*math.Pi*frequency*x[i]))
	}
	signal = &Signal{
		SampleRate: sampleRate,
		Samples:    y,
	}
}

func TestSignal_Num(t *testing.T) {
	assert.Equal(t, int(sampleRate*duration), signal.Num())
}

func TestSignal_Duration(t *testing.T) {
	assert.Equal(t, duration, signal.Duration(), epsilon)
}

func TestSignal_Truncate(t *testing.T) {
	s := &Signal{
		SampleRate: float64(1) / float64(60),
		Samples:    make([]float64, 60*24*15),
	}
	s, m := s.Truncate(time.Hour * 24 * 7)
	assert.Equal(t, 2, m)
	assert.Equal(t, 60*24*14, len(s.Samples))
}

func TestSignal_Min(t *testing.T) {
	assert.InEpsilon(t, -amplitude, signal.Min(), epsilon)
}

func TestSignal_Max(t *testing.T) {
	assert.InEpsilon(t, amplitude, signal.Max(), epsilon)
}

func TestSignal_Normalize(t *testing.T) {
	normalized, err := signal.Normalize()
	assert.NoError(t, err)
	assert.InEpsilon(t, -1.0, normalized.Min(), epsilon)
	assert.InEpsilon(t, 1.0, normalized.Max(), epsilon)
}

func TestSignal_Denormalize(t *testing.T) {
	min, max := signal.Min(), signal.Max()
	normalized, err := signal.Normalize()
	assert.NoError(t, err)
	denormalized, err := normalized.Denormalize(min, max)
	assert.NoError(t, err)
	assert.InEpsilonSlice(t, signal.Samples, denormalized.Samples, amplitude*0.01)
}

func TestSignal_FindPeriod(t *testing.T) {
	signals := make([]*Signal, nInputs)
	for i := 0; i < nInputs; i++ {
		s, err := readCsvFile(fmt.Sprintf("test_data/input%d.csv", i))
		assert.NoError(t, err)
		s, _ = s.Truncate(Week)
		signals[i] = s
	}

	signals[15].SampleRate = 1. / 15.

	periods := []time.Duration{
		Day,
		Day,
		Week,
		-1,
		-1,
		-1,
		-1,
		-1,
		Week,
		Day,
		Week,
		Day, // i: 11
		Week,
		Week,
		Week,
		Week,
	}

	for i := 0; i < nInputs; i++ {
		if periodic[i] {
			assert.Equal(t, periods[i], signals[i].FindPeriod(), "i: %d", i)
		}
	}
}

// Uncomment test below to see what time series in test_data look like.
// "Green" charts are  periodic time series, and "red" charts represent the non-periodic.
//func TestSignal_Plot(t *testing.T) {
//	signals := make([]*Signal, nInputs)
//	for i := 0; i < nInputs; i++ {
//		s, err := readCsvFile(fmt.Sprintf("test_data/input%d.csv", i))
//		assert.NoError(t, err)
//		signals[i] = s
//	}
//
//	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
//		p := components.NewPage()
//		for i := 0; i < nInputs; i++ {
//			var o charts.GlobalOpts
//			if periodic[i] {
//				o = charts.WithInitializationOpts(opts.Initialization{Width: "2000px", Theme: types.ThemeWonderland})
//			} else {
//				o = charts.WithInitializationOpts(opts.Initialization{Width: "2000px", Theme: types.ThemeRoma})
//			}
//			p = p.AddCharts(signals[i].Plot("", o))
//		}
//		_ = p.Render(w)
//	})
//	fmt.Println("Open your browser and access 'http://localhost:7001'")
//	_ = http.ListenAndServe(":7001", nil)
//}

var periodic = []bool{
	true,
	true,
	true,
	false, //true,
	false,
	false,
	false,
	false,
	true,
	true,
	true,
	true,
	true,
	true,
	true,
	true,
}

func readCsvFile(filename string) (*Signal, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(bytes.NewBuffer(buf))
	records, _ := reader.ReadAll()
	var values []float64
	for i := 1; i < len(records); i++ {
		val, _ := strconv.ParseFloat(records[i][1], 64)
		values = append(values, val)

	}
	return &Signal{
		SampleRate: 1.0 / 60.0,
		Samples:    values,
	}, nil
}
