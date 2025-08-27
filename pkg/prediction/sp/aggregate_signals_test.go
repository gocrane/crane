package sp

import (
	"fmt"
	predconf "github.com/gocrane/crane/pkg/prediction/config"
	"math"
	"runtime"
	"strconv"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/labels"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/metricquery"
	"github.com/gocrane/crane/pkg/prediction"
	"github.com/stretchr/testify/assert"
)

func TestAggregateSignals_Add(t *testing.T) {
	a := newAggregateSignals()
	namerHello := &metricnaming.GeneralMetricNamer{
		Metric: &metricquery.Metric{
			Type: metricquery.PromQLMetricType,
			Prom: &metricquery.PromNamerInfo{
				QueryExpr: "hello",
				Selector:  labels.Nothing(),
			},
		}}
	qc := prediction.QueryExprWithCaller{
		MetricNamer: namerHello,
		Caller:      "link",
	}
	assert.True(t, a.Add(qc))
	assert.False(t, a.Add(qc))

	qc.Caller = "snake"
	assert.False(t, a.Add(qc))

	namerHey := &metricnaming.GeneralMetricNamer{
		Metric: &metricquery.Metric{
			Type: metricquery.PromQLMetricType,
			Prom: &metricquery.PromNamerInfo{
				QueryExpr: "hey",
				Selector:  labels.Nothing(),
			},
		}}
	qc.MetricNamer = namerHey
	assert.True(t, a.Add(qc))

	qc.Caller = "link"
	assert.False(t, a.Add(qc))
}

func TestAggregateSignals_Delete(t *testing.T) {
	namerHello := &metricnaming.GeneralMetricNamer{
		Metric: &metricquery.Metric{
			Type: metricquery.PromQLMetricType,
			Prom: &metricquery.PromNamerInfo{
				QueryExpr: "hello",
				Selector:  labels.Nothing(),
			},
		}}
	namerHey := &metricnaming.GeneralMetricNamer{
		Metric: &metricquery.Metric{
			Type: metricquery.PromQLMetricType,
			Prom: &metricquery.PromNamerInfo{
				QueryExpr: "hey",
				Selector:  labels.Nothing(),
			},
		}}

	a := newAggregateSignals()
	qc := prediction.QueryExprWithCaller{
		MetricNamer: namerHello,
		Caller:      "link",
	}
	a.Add(qc)
	qc.MetricNamer = namerHey
	a.Add(qc)
	qc.Caller = "snake"
	a.Add(qc)

	signal := &aggregateSignal{}
	a.SetSignal(namerHello.BuildUniqueKey(), "k1", signal)
	signal2 := &aggregateSignal{}
	a.SetSignal(namerHey.BuildUniqueKey(), "k1", signal2)

	qc = prediction.QueryExprWithCaller{
		MetricNamer: namerHello,
		Caller:      "link",
	}
	assert.True(t, a.Delete(qc))
	assert.Nil(t, a.GetSignal(namerHello.BuildUniqueKey(), "k1"))

	qc.MetricNamer = namerHey
	assert.False(t, a.Delete(qc))
	assert.NotNil(t, a.GetSignal(namerHey.BuildUniqueKey(), "k1"))
	assert.Equal(t, signal2, a.GetSignal(namerHey.BuildUniqueKey(), "k1"))

	qc.Caller = "snake"
	assert.True(t, a.Delete(qc))
	assert.Nil(t, a.GetSignal("key", "k1"))
}

func TestAggregateSignals_SetSignal(t *testing.T) {
	namerHello := &metricnaming.GeneralMetricNamer{
		Metric: &metricquery.Metric{
			Type: metricquery.PromQLMetricType,
			Prom: &metricquery.PromNamerInfo{
				QueryExpr: "hello",
				Selector:  labels.Nothing(),
			},
		}}
	queryExpr := namerHello.BuildUniqueKey()
	a := newAggregateSignals()
	signal := &aggregateSignal{}
	a.SetSignal(queryExpr, "link", signal)
	assert.Nil(t, a.GetSignal(queryExpr, "link"))
	a.Add(prediction.QueryExprWithCaller{
		MetricNamer: namerHello,
		Caller:      "link",
	})
	a.SetSignal(queryExpr, "link", signal)
	assert.Equal(t, signal, a.GetSignal(queryExpr, "link"))
}

func TestAggregateSignals_GetSignal(t *testing.T) {
	namerHello := &metricnaming.GeneralMetricNamer{
		Metric: &metricquery.Metric{
			Type: metricquery.PromQLMetricType,
			Prom: &metricquery.PromNamerInfo{
				QueryExpr: "hello",
				Selector:  labels.Nothing(),
			},
		}}
	queryExpr := namerHello.BuildUniqueKey()

	a := newAggregateSignals()
	a.Add(prediction.QueryExprWithCaller{
		MetricNamer: namerHello,
		Caller:      "link",
	})
	signal := &aggregateSignal{}
	a.SetSignal(queryExpr, "k1", signal)
	assert.Equal(t, signal, a.GetSignal(queryExpr, "k1"))

	signal2 := &aggregateSignal{}
	a.SetSignal(queryExpr, "k2", signal2)
	assert.Equal(t, signal2, a.GetSignal(queryExpr, "k2"))
}

func TestAggregateSignals_GetOrStoreSignal(t *testing.T) {
	namerHello := &metricnaming.GeneralMetricNamer{
		Metric: &metricquery.Metric{
			Type: metricquery.PromQLMetricType,
			Prom: &metricquery.PromNamerInfo{
				QueryExpr: "hello",
				Selector:  labels.Nothing(),
			},
		}}
	queryExpr := namerHello.BuildUniqueKey()

	a := newAggregateSignals()
	qc := prediction.QueryExprWithCaller{
		MetricNamer: namerHello,
		Caller:      "link",
	}
	a.Add(qc)
	signal := &aggregateSignal{}
	a.SetSignal(queryExpr, "k1", signal)
	assert.Equal(t, signal, a.GetOrStoreSignal(queryExpr, "k1", signal))
	assert.Equal(t, signal, a.GetSignal(queryExpr, "k1"))
}

// TestAggregateSignals_MemoryUsage records the memory and time overhead of creating, deleting, and storing signals
func TestAggregateSignals_MemoryUsage(t *testing.T) {
	a := newAggregateSignals()

	// Get initial memory status
	var m1, m2, m3 runtime.MemStats
	runtime.GC() // Warm up GC
	runtime.GC() // Ensure initial state is clean
	runtime.ReadMemStats(&m1)
	initialMallocs := m1.Mallocs

	// Add a large number of signals
	for i := 0; i < 1000; i++ {
		namer := &metricnaming.GeneralMetricNamer{
			Metric: &metricquery.Metric{
				Type: metricquery.PromQLMetricType,
				Prom: &metricquery.PromNamerInfo{
					QueryExpr: "memory_usage_" + strconv.Itoa(i),
					Selector:  labels.Nothing(),
				},
			}}
		qc := prediction.QueryExprWithCaller{
			MetricNamer: namer,
			Caller:      "memory_test",
		}
		a.Add(qc)
		a.SetSignal(namer.BuildUniqueKey(), "memory_test", &aggregateSignal{})
	}

	// Record memory status after object creation
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Delete all signals
	for i := 0; i < 1000; i++ {
		namer := &metricnaming.GeneralMetricNamer{
			Metric: &metricquery.Metric{
				Type: metricquery.PromQLMetricType,
				Prom: &metricquery.PromNamerInfo{
					QueryExpr: "memory_usage_" + strconv.Itoa(i),
					Selector:  labels.Nothing(),
				},
			}}
		qc := prediction.QueryExprWithCaller{
			MetricNamer: namer,
			Caller:      "memory_test",
		}
		a.Delete(qc)
	}

	// Get final memory status
	runtime.GC()
	runtime.ReadMemStats(&m3)
	finalMallocs := m3.Mallocs

	// Calculate memory change
	createdMem := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	finalMem := int64(m3.HeapAlloc) - int64(m1.HeapAlloc)
	allocCount := finalMallocs - initialMallocs

	// Output detailed memory information
	t.Logf("Initial heap memory: %.2f MB, object count: %d", float64(m1.HeapAlloc)/1024/1024, m1.HeapObjects)
	t.Logf("Heap memory after object creation: %.2f MB, object count: %d", float64(m2.HeapAlloc)/1024/1024, m2.HeapObjects)
	t.Logf("Final heap memory: %.2f MB, object count: %d", float64(m3.HeapAlloc)/1024/1024, m3.HeapObjects)
	t.Logf("Memory increment after object creation: %.2f MB", float64(createdMem)/1024/1024)
	t.Logf("Final memory increment: %.2f MB", float64(finalMem)/1024/1024)
	t.Logf("Total allocation count: %d", allocCount)

	// Verify whether memory basically returns to the initial state
	memoryDelta := math.Abs(float64(finalMem))
	if memoryDelta > float64(10*1024) { // Allow 10KB difference
		t.Errorf("Memory not fully released, difference: %.2f KB", memoryDelta/1024)
	} else {
		t.Logf("Memory basically released, difference: %.2f KB", memoryDelta/1024)
	}
}

// TestAggregateSignalsDetails_MemoryUsage records the memory and time overhead of creating detailed signals, i.e., storing collected information in 24 histograms
func TestAggregateSignalsDetails_MemoryUsage(t *testing.T) {
	const (
		tspNum         = 190
		historyLength  = "168h"
		sampleInterval = "5m"
	)

	a := newAggregateSignals()
	cfg := predconf.Config{
		SP: &predictionapi.SP{
			HistoryLength:  historyLength,
			SampleInterval: sampleInterval,
			Histogram: predictionapi.HistogramConfig{
				MaxValue:   "10.0",
				Epsilon:    "1e-10",
				HalfLife:   "0h",
				BucketSize: "0.01",
			},
		},
	}
	ts := generateFakeTimeSeriesByHour(historyLength, sampleInterval)
	historyTimeSeriesList := []*common.TimeSeries{ts}

	// Get initial memory status
	var m1, m2, m3 runtime.MemStats
	runtime.GC() // Warm up GC
	runtime.GC() // Ensure initial state is clean
	runtime.ReadMemStats(&m1)
	initialMallocs := m1.Mallocs

	// Record time
	startTime := time.Now()

	// Add a large number of signals
	for i := 0; i < tspNum; i++ {
		namer := &metricnaming.GeneralMetricNamer{
			Metric: &metricquery.Metric{
				Type: metricquery.PromQLMetricType,
				Prom: &metricquery.PromNamerInfo{
					QueryExpr: "memory_usage_" + strconv.Itoa(i),
					Selector:  labels.Nothing(),
				},
			}}
		qc := prediction.QueryExprWithCaller{
			MetricNamer: namer,
			Config:      cfg,
			Caller:      "memory_test",
		}
		a.Add(qc)
		queryExpr := namer.BuildUniqueKey()
		icfg := a.GetConfig(queryExpr)
		signals := getSignalsFromTS(historyTimeSeriesList, icfg)
		a.SetSignals(queryExpr, signals)
	}

	// Record memory status after object creation
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Delete all signals
	for i := 0; i < tspNum; i++ {
		namer := &metricnaming.GeneralMetricNamer{
			Metric: &metricquery.Metric{
				Type: metricquery.PromQLMetricType,
				Prom: &metricquery.PromNamerInfo{
					QueryExpr: "memory_usage_" + strconv.Itoa(i),
					Selector:  labels.Nothing(),
				},
			}}
		qc := prediction.QueryExprWithCaller{
			MetricNamer: namer,
			Config:      cfg,
			Caller:      "memory_test",
		}
		a.Delete(qc)
	}

	// Record time
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	t.Logf("Time taken to create and delete signals: %.2f seconds", duration.Seconds())

	// Get final memory status
	runtime.GC()
	runtime.ReadMemStats(&m3)
	finalMallocs := m3.Mallocs

	// Calculate memory change
	createdMem := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	finalMem := int64(m3.HeapAlloc) - int64(m1.HeapAlloc)
	allocCount := finalMallocs - initialMallocs

	// Output detailed memory information
	t.Logf("Initial heap memory: %.2f MB, object count: %d", float64(m1.HeapAlloc)/1024/1024, m1.HeapObjects)
	t.Logf("Heap memory after object creation: %.2f MB, object count: %d", float64(m2.HeapAlloc)/1024/1024, m2.HeapObjects)
	t.Logf("Final heap memory: %.2f MB, object count: %d", float64(m3.HeapAlloc)/1024/1024, m3.HeapObjects)
	t.Logf("Memory increment after object creation: %.2f MB", float64(createdMem)/1024/1024)
	t.Logf("Final memory increment: %.2f MB", float64(finalMem)/1024/1024)
	t.Logf("Total allocation count: %d", allocCount)

	// Verify whether memory basically returns to the initial state
	if float64(finalMem) > float64(10*1024) { // Allow 10KB difference
		t.Errorf("Memory not fully released, difference: %.2f KB", float64(finalMem)/1024)
	} else {
		t.Logf("Memory basically released, difference: %.2f KB", float64(finalMem)/1024)
	}
}

// TestAggregateSignalsDetailsList_MemoryUsage records the change in memory and time as tspNum changes
func TestAggregateSignalsDetailsList_MemoryUsage(t *testing.T) {
	const (
		historyLength  = "168h"
		sampleInterval = "5m"
	)
	cfg := predconf.Config{
		SP: &predictionapi.SP{
			HistoryLength:  historyLength,
			SampleInterval: sampleInterval,
			Histogram: predictionapi.HistogramConfig{
				MaxValue:   "10.0",
				Epsilon:    "1e-10",
				HalfLife:   "0h",
				BucketSize: "0.01",
			},
		},
	}
	ts := generateFakeTimeSeriesByHour(historyLength, sampleInterval)
	historyTimeSeriesList := []*common.TimeSeries{ts}
	tspNumSlice := []int{}
	timeSlice := []float64{}
	memSlice := []float64{}

	for tspNum := 1; tspNum <= 2001; tspNum += 100 {
		a := newAggregateSignals()

		// Get initial memory status
		var m1, m2, m3 runtime.MemStats
		runtime.GC() // Warm up GC
		runtime.GC() // Ensure initial state is clean
		runtime.ReadMemStats(&m1)

		// Record time
		startTime := time.Now()

		// Add a large number of signals
		for i := 0; i < tspNum; i++ {
			namer := &metricnaming.GeneralMetricNamer{
				Metric: &metricquery.Metric{
					Type: metricquery.PromQLMetricType,
					Prom: &metricquery.PromNamerInfo{
						QueryExpr: "memory_usage_" + strconv.Itoa(i),
						Selector:  labels.Nothing(),
					},
				}}
			qc := prediction.QueryExprWithCaller{
				MetricNamer: namer,
				Config:      cfg,
				Caller:      "memory_test",
			}
			a.Add(qc)
			queryExpr := namer.BuildUniqueKey()
			icfg := a.GetConfig(queryExpr)
			signals := getSignalsFromTS(historyTimeSeriesList, icfg)
			a.SetSignals(queryExpr, signals)
		}

		// Record memory status after object creation
		runtime.GC()
		runtime.ReadMemStats(&m2)

		// Delete all signals
		for i := 0; i < tspNum; i++ {
			namer := &metricnaming.GeneralMetricNamer{
				Metric: &metricquery.Metric{
					Type: metricquery.PromQLMetricType,
					Prom: &metricquery.PromNamerInfo{
						QueryExpr: "memory_usage_" + strconv.Itoa(i),
						Selector:  labels.Nothing(),
					},
				}}
			qc := prediction.QueryExprWithCaller{
				MetricNamer: namer,
				Config:      cfg,
				Caller:      "memory_test",
			}
			a.Delete(qc)
		}

		// Record time
		endTime := time.Now()
		duration := endTime.Sub(startTime)

		// Get final memory status
		runtime.GC()
		runtime.ReadMemStats(&m3)

		// Calculate memory change
		createdMem := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)

		tspNumSlice = append(tspNumSlice, tspNum)
		timeSlice = append(timeSlice, duration.Seconds())
		memSlice = append(memSlice, float64(createdMem)/1024/1024)
	}

	fmt.Printf("tsp_num = [")
	for i := 0; i < len(tspNumSlice); i++ {
		fmt.Printf("%v", tspNumSlice[i])
		if i != len(tspNumSlice)-1 {
			fmt.Printf(", ")
		} else {
			fmt.Println("]")
		}
	}
	fmt.Printf("time_list = [") // seconds
	for i := 0; i < len(timeSlice); i++ {
		fmt.Printf("%v", timeSlice[i])
		if i != len(tspNumSlice)-1 {
			fmt.Printf(", ")
		} else {
			fmt.Println("]")
		}
	}
	fmt.Printf("mem_list = [") // MB
	for i := 0; i < len(memSlice); i++ {
		fmt.Printf("%v", memSlice[i])
		if i != len(tspNumSlice)-1 {
			fmt.Printf(", ")
		} else {
			fmt.Println("]")
		}
	}
}

func timeString2Int(t string) int {
	unit := t[len(t)-1]
	valueStr := t[:len(t)-1]
	value, err := strconv.Atoi(valueStr)
	tMinutes := 0
	if err == nil {
		switch unit {
		case 'm', 'M':
			tMinutes = value
		case 'h', 'H':
			tMinutes = value * 60
		case 'd', 'D':
			tMinutes = value * 24 * 60
		}
	}
	return tMinutes
}

func generateFakeTimeSeriesByHour(historyLength, sampleInterval string) *common.TimeSeries {
	historyMinutes := timeString2Int(historyLength)
	sampleMinutes := timeString2Int(sampleInterval)
	samplesLength := historyMinutes / sampleMinutes

	return generateFakeTimeSeriesByNum(samplesLength, sampleMinutes)
}

// generateFakeTimeSeries generates a fake TimeSeries with specified number of samples
func generateFakeTimeSeriesByNum(samplesLength int, sampleInterval int) *common.TimeSeries {
	if samplesLength <= 0 {
		samplesLength = 1
	}

	ts := &common.TimeSeries{}

	// Add fake labels
	ts.Labels = []common.Label{}
	baseTimestamp := int64(0)
	for i := 0; i < samplesLength; i++ {
		timestamp := baseTimestamp + int64(i*60000*sampleInterval)
		value := 1.0
		ts.Samples = append(ts.Samples, common.Sample{
			Timestamp: timestamp,
			Value:     value,
		})
	}

	return ts
}
