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

// TestAggregateSignals_MemoryUsage 记录创建、删除和存储信号的内存和时间开销
func TestAggregateSignals_MemoryUsage(t *testing.T) {
	a := newAggregateSignals()

	// 获取初始内存状态
	var m1, m2, m3 runtime.MemStats
	runtime.GC() // 预热GC
	runtime.GC() // 确保初始状态干净
	runtime.ReadMemStats(&m1)
	initialMallocs := m1.Mallocs

	// 添加大量信号
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

	// 记录对象创建后的内存状态
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// 删除所有信号
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

	// 获取最终内存状态
	runtime.GC()
	runtime.ReadMemStats(&m3)
	finalMallocs := m3.Mallocs

	// 计算内存变化
	createdMem := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	finalMem := int64(m3.HeapAlloc) - int64(m1.HeapAlloc)
	allocCount := finalMallocs - initialMallocs

	// 输出详细内存信息
	t.Logf("初始堆内存: %.2f MB, 对象数: %d", float64(m1.HeapAlloc)/1024/1024, m1.HeapObjects)
	t.Logf("创建对象后堆内存: %.2f MB, 对象数: %d", float64(m2.HeapAlloc)/1024/1024, m2.HeapObjects)
	t.Logf("最终堆内存: %.2f MB, 对象数: %d", float64(m3.HeapAlloc)/1024/1024, m3.HeapObjects)
	t.Logf("创建对象内存增量: %.2f MB", float64(createdMem)/1024/1024)
	t.Logf("最终内存增量: %.2f MB", float64(finalMem)/1024/1024)
	t.Logf("总分配次数: %d", allocCount)

	// 验证内存是否基本回到初始状态
	memoryDelta := math.Abs(float64(finalMem))
	if memoryDelta > float64(10*1024) { // 允许10KB差异
		t.Errorf("内存未完全释放，差异: %.2f KB", memoryDelta/1024)
	} else {
		t.Logf("内存基本释放，差异: %.2f KB", memoryDelta/1024)
	}
}

// TestAggregateSignalsDetails_MemoryUsage 记录创建详细的signals的内存和时间开销，即将采集到的信息存储在24个直方图中
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

	// 获取初始内存状态
	var m1, m2, m3 runtime.MemStats
	runtime.GC() // 预热GC
	runtime.GC() // 确保初始状态干净
	runtime.ReadMemStats(&m1)
	initialMallocs := m1.Mallocs

	// 记录时间
	startTime := time.Now()

	// 添加大量信号
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

	// 记录对象创建后的内存状态
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// 删除所有信号
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

	// 记录时间
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	t.Logf("创建和删除信号耗时: %.2f 秒", duration.Seconds())

	// 获取最终内存状态
	runtime.GC()
	runtime.ReadMemStats(&m3)
	finalMallocs := m3.Mallocs

	// 计算内存变化
	createdMem := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	finalMem := int64(m3.HeapAlloc) - int64(m1.HeapAlloc)
	allocCount := finalMallocs - initialMallocs

	// 输出详细内存信息
	t.Logf("初始堆内存: %.2f MB, 对象数: %d", float64(m1.HeapAlloc)/1024/1024, m1.HeapObjects)
	t.Logf("创建对象后堆内存: %.2f MB, 对象数: %d", float64(m2.HeapAlloc)/1024/1024, m2.HeapObjects)
	t.Logf("最终堆内存: %.2f MB, 对象数: %d", float64(m3.HeapAlloc)/1024/1024, m3.HeapObjects)
	t.Logf("创建对象内存增量: %.2f MB", float64(createdMem)/1024/1024)
	t.Logf("最终内存增量: %.2f MB", float64(finalMem)/1024/1024)
	t.Logf("总分配次数: %d", allocCount)

	// 验证内存是否基本回到初始状态
	if float64(finalMem) > float64(10*1024) { // 允许增量10KB差异
		t.Errorf("内存未完全释放，差异: %.2f KB", float64(finalMem)/1024)
	} else {
		t.Logf("内存基本释放，差异: %.2f KB", float64(finalMem)/1024)
	}
}

// TestAggregateSignalsDetailsList_MemoryUsage 记录tspNum变化时，内存和时间的变化
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

		// 获取初始内存状态
		var m1, m2, m3 runtime.MemStats
		runtime.GC() // 预热GC
		runtime.GC() // 确保初始状态干净
		runtime.ReadMemStats(&m1)

		// 记录时间
		startTime := time.Now()

		// 添加大量信号
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

		// 记录对象创建后的内存状态
		runtime.GC()
		runtime.ReadMemStats(&m2)

		// 删除所有信号
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

		// 记录时间
		endTime := time.Now()
		duration := endTime.Sub(startTime)

		// 获取最终内存状态
		runtime.GC()
		runtime.ReadMemStats(&m3)

		// 计算内存变化
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
