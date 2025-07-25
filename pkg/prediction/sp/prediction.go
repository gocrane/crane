package sp

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/prediction"
	"github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/providers"
)

var _ prediction.Interface = &segmentedPercentilePrediction{}
var keyAll = "__all__"

type segmentedPercentilePrediction struct {
	prediction.GenericPrediction
	a aggregateSignals
	// record the query routine already started
	queryRoutines sync.Map
	stopChMap     sync.Map
}

func (p *segmentedPercentilePrediction) QueryPredictionStatus(_ context.Context, metricNamer metricnaming.MetricNamer) (prediction.Status, error) {
	_, status := p.a.GetSignals(metricNamer.BuildUniqueKey())
	return status, nil
}

func (p *segmentedPercentilePrediction) QueryPredictedTimeSeries(ctx context.Context, namer metricnaming.MetricNamer, startTime time.Time, endTime time.Time) ([]*common.TimeSeries, error) {
	testStartTime := time.Now()
	defer func() {
		klog.V(1).Infof("Finished QueryPredictedTimeSeries (%v)", time.Since(testStartTime))
	}()
	var predictedTimeSeriesList []*common.TimeSeries
	queryExpr := namer.BuildUniqueKey()
	//cfg := p.a.GetConfig(queryExpr)
	tsList := p.getPredictedValues(ctx, namer)
	klog.V(5).Infof("tsList len: %v, QueryPredictedTimeSeries queryExpr: %v", len(tsList), queryExpr)
	for _, ts := range tsList { // 理论上只会有一个ts，即len(tsList) == 1
		klog.V(5).Infof("ts label len: %v, ts sample len: %v, ts.Samples: %+v, QueryPredictedTimeSeries queryExpr: %v", len(ts.Labels), len(ts.Samples), ts.Samples, queryExpr)
		var samples []common.Sample
		sampleTime := startTime
		stepTime := time.Duration(0)
		if len(ts.Samples) > 1 {
			// 保证startTime+step*len(ts.Samples) >= endTime
			stepTime = time.Duration(int64(endTime.Sub(startTime))/int64(len(ts.Samples)-1) + 1)
		}

		for _, s := range ts.Samples {
			//klog.V(1).Infof("ts sample: %+v, QueryPredictedTimeSeries queryExpr: %v", s, queryExpr)
			samples = append(samples, common.Sample{Timestamp: sampleTime.Unix(), Value: s.Value})
			sampleTime = sampleTime.Add(stepTime)
		}
		predictedTimeSeriesList = append(predictedTimeSeriesList, &common.TimeSeries{
			Labels:  ts.Labels,
			Samples: samples,
		})
	}
	return predictedTimeSeriesList, nil
}

func (p *segmentedPercentilePrediction) getPredictedValuesFromSignals(queryExpr string, signals map[string]*aggregateSignal, cfg *internalConfig) []*common.TimeSeries {
	startTime := time.Now()
	defer func() {
		klog.V(1).Infof("Finished getPredictedValuesFromSignals (%v) queryExpr:%v", time.Since(startTime), queryExpr)
	}()
	var predictedTimeSeriesList []*common.TimeSeries

	if cfg == nil {
		cfg = p.a.GetConfig(queryExpr)
	}
	estimator := NewPercentileEstimator(cfg.percentile)
	estimator = WithMargin(cfg.marginFraction, estimator)
	estimator = WithTargetUtilization(cfg.targetUtilization, estimator)
	now := time.Now().Unix()
	klog.V(4).Infof("getPredictedValuesFromSignals now: %v, queryExpr: %v", now, queryExpr)
	if cfg.aggregated {
		signal := signals[keyAll]
		if signal != nil {
			sample := common.Sample{
				Value:     estimator.GetEstimation(signal.histogram),
				Timestamp: now,
			}
			predictedTimeSeriesList = append(predictedTimeSeriesList, &common.TimeSeries{
				Labels:  nil,
				Samples: []common.Sample{sample},
			})
		}
	} else {
		// 提取所有键并排序
		keys := make([]string, 0, len(signals))
		for key := range signals {
			if key == keyAll {
				continue
			}
			keys = append(keys, key)
		}
		// 根据字符串中最后的数字部分排序
		sort.Slice(keys, func(i, j int) bool {
			// 提取数字部分
			numI := strings.Split(keys[i], "-")[1]
			numJ := strings.Split(keys[j], "-")[1]
			// 转换为整数比较
			intI, _ := strconv.Atoi(numI)
			intJ, _ := strconv.Atoi(numJ)
			return intI < intJ
		})

		samples := make([]common.Sample, 0, len(signals))
		labels := []common.Label{}
		// 按排序后的键顺序处理
		for _, key := range keys {
			signal := signals[key]
			klog.V(5).Infof("getPredictedValuesFromSignals key: %v, v: %v, P99:%v, now: %v, queryExpr: %v", key, estimator.GetEstimation(signal.histogram), signal.histogram.Percentile(99), now, queryExpr)
			sample := common.Sample{
				Value: signal.histogram.Percentile(99),
				//Value:     estimator.GetEstimation(signal.histogram),
				Timestamp: now,
			}
			samples = append(samples, sample)
			labels = append(labels, signal.labels...)
		}
		// 分管不同片的signal合并为一个时间序列
		predictedTimeSeriesList = append(predictedTimeSeriesList, &common.TimeSeries{
			Labels:  labels,
			Samples: samples,
		})
	}

	return predictedTimeSeriesList
}

func (p *segmentedPercentilePrediction) getPredictedValues(ctx context.Context, namer metricnaming.MetricNamer) []*common.TimeSeries {
	startTime := time.Now()
	defer func() {
		klog.V(1).Infof("Finished getPredictedValues (%v)", time.Since(startTime))
	}()
	var predictedTimeSeriesList []*common.TimeSeries

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	queryExpr := namer.BuildUniqueKey()
	for {
		signals, status := p.a.GetSignals(queryExpr)
		if status == prediction.StatusUnknown {
			klog.V(4).InfoS("Aggregated has been deleted and unknown", "queryExpr", queryExpr)
			return predictedTimeSeriesList
		}
		if signals != nil && status == prediction.StatusReady {
			klog.V(4).Infof("signals Ready getPredictedValues queryExpr: %v", queryExpr)
			return p.getPredictedValuesFromSignals(queryExpr, signals, nil)
		}
		select {
		case <-ctx.Done():
			klog.Info("Time out.")
			return predictedTimeSeriesList
		case <-ticker.C:
			continue
		}
	}
}

func (p *segmentedPercentilePrediction) QueryRealtimePredictedValues(ctx context.Context, namer metricnaming.MetricNamer) ([]*common.TimeSeries, error) {
	queryExpr := namer.BuildUniqueKey()
	_, status := p.a.GetSignals(queryExpr)
	if status != prediction.StatusReady {
		return nil, fmt.Errorf("metric %v model status is %v, must be ready", queryExpr, status)
	}
	return p.getPredictedValues(ctx, namer), nil
}

// QueryRealtimePredictedValuesOnce is a one-off task, and the query will be deleted by the caller after the call. However, this query maybe has already been used by other callers,
// if so, we get the estimated value from the model directly.
// When the query is not found, we fetch the history data to build the histogram, and then get the estimated value by a stateless function as data processing way.
func (p *segmentedPercentilePrediction) QueryRealtimePredictedValuesOnce(_ context.Context, namer metricnaming.MetricNamer, config config.Config) ([]*common.TimeSeries, error) {
	klog.V(1).Infof("QueryRealtimePredictedValuesOnce namer: %v", namer.BuildUniqueKey())
	queryExpr := namer.BuildUniqueKey()

	cfg, err := makeInternalConfig(config.SP, config.InitMode)
	if err != nil {
		return nil, err
	}

	signals, status := p.a.GetSignals(queryExpr)
	if signals != nil && status == prediction.StatusReady {
		return p.getPredictedValuesFromSignals(queryExpr, signals, cfg), nil
	} else {
		// namer metric query is firstly registered by this caller
		// we first fetch history data to construct the histogram model, then get estimation.
		// it is just a stateless function, a data analyzing process, data in, then data out, no states.
		return p.process(namer, cfg)
	}
}

// process is a stateless function to get estimation of a metric series by constructing a histogram then get estimation data.
func (p *segmentedPercentilePrediction) process(namer metricnaming.MetricNamer, cfg *internalConfig) ([]*common.TimeSeries, error) {
	var historyTimeSeriesList []*common.TimeSeries
	var err error
	queryExpr := namer.BuildUniqueKey()

	klog.V(4).Infof("process analyzing metric namer: %v, config: %+v", namer.BuildUniqueKey(), *cfg)

	historyTimeSeriesList, err = p.queryHistoryTimeSeries(namer, cfg)
	if err != nil {
		klog.Errorf("Failed to query history time series for query expression '%s'.", queryExpr)
		return nil, err
	}

	signals := map[string]*aggregateSignal{}
	if cfg.aggregated {
		signal := newAggregateSignal(cfg)
		for _, ts := range historyTimeSeriesList {
			for _, s := range ts.Samples {
				t := time.Unix(s.Timestamp, 0)
				signal.addSample(t, s.Value)
			}
		}
		signals[keyAll] = signal
	} else {
		for _, ts := range historyTimeSeriesList {
			if len(ts.Samples) < 1 {
				continue
			}
			key := prediction.AggregateSignalKey(ts.Labels)
			signal := newAggregateSignal(cfg)
			for _, s := range ts.Samples {
				t := time.Unix(s.Timestamp, 0)
				signal.addSample(t, s.Value)
			}
			signal.labels = ts.Labels
			signals[key] = signal
		}
	}

	return p.getPredictedValuesFromSignals(queryExpr, signals, cfg), nil
}

func NewPrediction(realtimeProvider providers.RealTime, historyProvider providers.History) prediction.Interface {
	withCh, delCh := make(chan prediction.QueryExprWithCaller), make(chan prediction.QueryExprWithCaller)
	return &segmentedPercentilePrediction{
		GenericPrediction: prediction.NewGenericPrediction(realtimeProvider, historyProvider, withCh, delCh),
		a:                 newAggregateSignals(),
		queryRoutines:     sync.Map{},
		stopChMap:         sync.Map{},
	}
}

func (p *segmentedPercentilePrediction) Run(stopCh <-chan struct{}) {
	// 处理注册操作
	go func() {
		for {
			qc := <-p.WithCh
			// update if the query config updated, idempotent
			p.a.Add(qc)
			klog.V(4).Infof("sp get queryExpr: %v", qc.MetricNamer.BuildUniqueKey())
			QueryExpr := qc.MetricNamer.BuildUniqueKey()

			if _, ok := p.queryRoutines.Load(QueryExpr); ok {
				klog.V(4).InfoS("Prediction segmented percentile routine already registered.", "queryExpr", QueryExpr, "caller", qc.Caller)
				continue
			}

			if _, ok := p.stopChMap.Load(QueryExpr); ok {
				klog.V(4).InfoS("Prediction segmented percentile routine already stopped.", "queryExpr", QueryExpr, "caller", qc.Caller)
				continue
			}

			klog.V(4).InfoS("Register a query expression for prediction.", "queryExpr", QueryExpr, "caller", qc.Caller)
			// todo: Do not block this management go routine here to do some time consuming operation.
			// We just init the signal and setting the status
			// we start the real time model updating directly. but there is a window time for each metricNamer in the algorithm config to ready status
			c := p.a.GetConfig(QueryExpr)

			var initError error
			switch c.initMode {
			case config.ModelInitModeLazyTraining:
				p.initByRealTimeProvider(qc.MetricNamer)
			case config.ModelInitModeCheckpoint:
				initError = p.initByCheckPoint(qc.MetricNamer)
			case config.ModelInitModeHistory:
				fallthrough
			default:
				// blocking
				initError = p.initFromHistory(qc.MetricNamer)
			}

			if initError != nil {
				klog.ErrorS(initError, "Failed to init segmentedPercentilePrediction.")
				continue
			}

			// 在goroutine之前注册，保证只注册一次
			queryExpr := qc.MetricNamer.BuildUniqueKey()
			p.queryRoutines.Store(queryExpr, struct{}{})

			// note: same query, but different config means different series analysis, GetConfig always return same config.
			// this is our default policy, one metric only has one config at a time.
			go func(namer metricnaming.MetricNamer) {
				if c := p.a.GetConfig(queryExpr); c != nil {
					ticker := time.NewTicker(c.sampleInterval)
					defer ticker.Stop()

					v, _ := p.stopChMap.LoadOrStore(queryExpr, make(chan struct{}))
					predStopCh := v.(chan struct{})

					for {
						// 每隔c.sampleInterval，就去获取最新数据，更新模型
						p.addSamples(namer)
						select {
						case <-predStopCh:
							p.queryRoutines.Delete(queryExpr)
							klog.V(4).InfoS("Prediction routine stopped.", "queryExpr", queryExpr)
							return
						case <-ticker.C:
							continue
						}
					}
				}
			}(qc.MetricNamer)
		}
	}()

	// 处理删除操作
	go func() {
		for {
			qc := <-p.DelCh
			QueryExpr := qc.MetricNamer.BuildUniqueKey()
			klog.V(4).InfoS("Unregister a query expression from prediction.", "queryExpr", QueryExpr, "caller", qc.Caller)

			go func(qc prediction.QueryExprWithCaller) {
				if p.a.Delete(qc) {
					val, loaded := p.stopChMap.LoadAndDelete(QueryExpr)
					if loaded {
						predStopCh := val.(chan struct{})
						predStopCh <- struct{}{}
					}
				}
			}(qc)
		}
	}()

	klog.Infof("predictor %v started", p.Name())

	<-stopCh

	klog.Infof("predictor %v stopped", p.Name())

}

func (p *segmentedPercentilePrediction) queryHistoryTimeSeries(namer metricnaming.MetricNamer, c *internalConfig) ([]*common.TimeSeries, error) {
	if p.GetHistoryProvider() == nil {
		klog.Fatalln("History provider not found")
	}
	end := time.Now().Truncate(time.Minute)
	start := end.Add(-c.historyLength)
	historyTimeSeries, err := p.GetHistoryProvider().QueryTimeSeries(namer, start, end, c.sampleInterval)
	if err != nil {
		klog.ErrorS(err, "Failed to query history time series.")
		return nil, err
	}
	return historyTimeSeries, nil
}

// Lazy training the histogram model. we do not init from History Provider such as prometheus because prometheus's poor performance issue.
// Rather, we recover or init a complete available histogram model by fetching real time data point continuously until it is enough time to do estimation.
// So, we can set a waiting time for the model trained completed. Because percentile is only used for request resource & resource estimation.
// Because of the scenes, we do not need it give a result fastly after service start, we can tolerate it has some days delaying for collecting more data.
// nolint:unused
func (p *segmentedPercentilePrediction) initByRealTimeProvider(namer metricnaming.MetricNamer) {
	queryExpr := namer.BuildUniqueKey()
	cfg := p.a.GetConfig(queryExpr)
	if cfg.aggregated {
		signal := newAggregateSignal(cfg)

		p.a.SetSignalWithStatus(queryExpr, keyAll, signal, prediction.StatusInitializing)
	} else {
		signals := map[string]*aggregateSignal{}
		p.a.SetSignalsWithStatus(queryExpr, signals, prediction.StatusInitializing)
	}
}

// todo:
// nolint:unused
func (p *segmentedPercentilePrediction) initByCheckPoint(_ metricnaming.MetricNamer) error {
	return fmt.Errorf("checkpoint not supported")
}

func (p *segmentedPercentilePrediction) initFromHistory(namer metricnaming.MetricNamer) error {
	startTime := time.Now()
	defer func() {
		klog.V(1).Infof("Finished initFromHistory (%v)", time.Since(startTime))
	}()
	queryExpr := namer.BuildUniqueKey()
	cfg := p.a.GetConfig(queryExpr)
	// Query history data for prediction
	var historyTimeSeriesList []*common.TimeSeries
	var err error
	historyTimeSeriesList, err = p.queryHistoryTimeSeries(namer, cfg)
	if err != nil {
		klog.Errorf("Failed to query history time series for query expression '%s'.", queryExpr)
		return err
	}
	klog.V(1).Infof("Got history time series, len: %v queryExpr: %v", len(historyTimeSeriesList), queryExpr)
	for _, ts := range historyTimeSeriesList {
		klog.V(1).Infof("Got history time series, labels: %v, samples: %v", len(ts.Labels), len(ts.Samples))
	}
	if cfg.aggregated {
		signal := newAggregateSignal(cfg)
		for _, ts := range historyTimeSeriesList {
			for _, s := range ts.Samples {
				t := time.Unix(s.Timestamp, 0)
				signal.addSample(t, s.Value)
			}
		}
		p.a.SetSignal(queryExpr, keyAll, signal)
	} else {
		signals := getSignalsFromTS(historyTimeSeriesList, cfg)
		// TODO 需要检测是否Ready
		p.a.SetSignals(queryExpr, signals)
	}

	return nil
}

func getSignalsFromTS(historyTimeSeriesList []*common.TimeSeries, cfg *internalConfig) map[string]*aggregateSignal {
	signals := map[string]*aggregateSignal{}
	for _, ts := range historyTimeSeriesList {
		if len(ts.Samples) < 1 {
			continue
		}
		key := prediction.AggregateSignalKey(ts.Labels)
		// 创建24个桶，每个桶存储一个小时的数据
		for i := 0; i < 24; i++ {
			timeKey := fmt.Sprintf("%s-%d", key, i)
			signal := newAggregateSignal(cfg)
			signals[timeKey] = signal
		}
		// 将数据按照小时存储到桶中
		for _, s := range ts.Samples {
			t := time.Unix(s.Timestamp, 0)
			timeKey := fmt.Sprintf("%s-%d", key, t.Hour())
			signals[timeKey].addSample(t, s.Value)
		}
		//for i := 0; i < 24; i++ {
		//	timeKey := fmt.Sprintf("%s-%d", key, i)
		//	klog.V(1).Infof("initFromHistory After timeKey:%v, P99:%v AggregationWindowLength: %v", timeKey, signals[timeKey].histogram.Percentile(99), signals[timeKey].GetAggregationWindowLength())
		//}
	}
	return signals
}

func (p *segmentedPercentilePrediction) addSamples(namer metricnaming.MetricNamer) {
	startTime := time.Now()
	defer func() {
		klog.V(1).Infof("Finished addSamples (%v)", time.Since(startTime))
	}()
	latestTimeSeriesList, err := p.GetRealtimeProvider().QueryLatestTimeSeries(namer)
	if err != nil {
		klog.ErrorS(err, "Failed to get latest time series.")
		return
	}

	queryExpr := namer.BuildUniqueKey()
	c := p.a.GetConfig(queryExpr)

	if c.aggregated {
		signal := p.a.GetSignal(queryExpr, keyAll)
		if signal == nil {
			return
		}
		// maybe we can use other aggregated way to deal with the container instances of the same container in different pods of the same workload,
		// aggregated by reducing all the samples to just one p99 or avg value and so on.
		// saw that when the workload is daemonset, container in different node has very different resource usage. this is unexpected in production, maybe the daemonset in different nodes has different loads.
		// NOTE: now it there are N instance of the workload, there are N samples in latest, then the aggregationWindowLength is N times growth to accumulate data fastly.
		// it is not a time dimension, but we use N samples of different container instances of the workload to represent the N intervals samples
		for _, ts := range latestTimeSeriesList {
			if len(ts.Samples) < 1 {
				klog.V(4).InfoS("Sample not found.", "key", keyAll)
				continue
			}
			sample := ts.Samples[len(ts.Samples)-1]
			sampleTime := time.Unix(sample.Timestamp, 0)
			signal.addSample(sampleTime, sample.Value)

			// current time is reach the window length of percentile need to accumulating data, the model is ready to do predict
			// all type need the aggregation window length is accumulating enough data.
			// History: if the container is newly, then there maybe has not enough history data, so we need accumulating to make the confidence more reliable
			// LazyTraining: directly accumulating data from real time metric provider until the data is enough
			// Checkpoint: directly recover the model from a checkpoint, and then updating the model until accumulated data is enough
			if signal.GetAggregationWindowLength() >= c.historyLength {
				p.a.SetSignalStatus(queryExpr, keyAll, prediction.StatusReady)
			}

			klog.V(4).InfoS("Sample added.", "sampleValue", sample.Value, "sampleTime", sampleTime, "queryExpr", queryExpr, "history", c.historyLength, "aggregationWindowLength", signal.GetAggregationWindowLength())
		}
	} else {
		// todo: find a way to remove the labels key, although we do not really use it now.
		for _, ts := range latestTimeSeriesList {
			key := prediction.AggregateSignalKey(ts.Labels)
			if len(ts.Samples) < 1 {
				klog.V(4).InfoS("Sample not found.", "queryExpr", queryExpr, "key", key)
				continue
			}
			sample := ts.Samples[len(ts.Samples)-1]
			sampleTime := time.Unix(sample.Timestamp, 0)
			keyTime := fmt.Sprintf("%s-%d", key, sampleTime.Hour())
			// todo: newAggregateSignal(c)延迟到查询不到再创建
			signal := p.a.GetOrStoreSignal(queryExpr, keyTime, newAggregateSignal(c))
			if signal == nil {
				continue
			}

			signal.addSample(sampleTime, sample.Value)
			//signal.addSample(sampleTime, float64((sampleTime.Hour()+1)*10))
			if len(signal.labels) == 0 {
				signal.labels = ts.Labels
			}

			// current time is reach the window length of percentile need to accumulating data, the model is ready to do predict
			// all type need the aggregation window length is accumulating enough data.
			// History: if the container is newly, then there maybe has not enough history data, so we need accumulating to make the confidence more reliable
			// LazyTraining: directly accumulating data from real time metric provider until the data is enough
			// Checkpoint: directly recover the model from a checkpoint, and then updating the model until accumulated data is enough
			if signal.GetAggregationWindowLength() >= c.historyLength {
				klog.V(1).Infof("signal[%v] Ready signal.GetAggregationWindowLength(): %v, historyLength: %v", key, signal.GetAggregationWindowLength(), c.historyLength)
				p.a.SetSignalStatus(queryExpr, key, prediction.StatusReady)
			}

			klog.V(4).InfoS("Sample added.", "sampleValue", sample.Value, "sampleTime", sampleTime, "queryExpr", queryExpr, "key", key, "history", c.historyLength, "aggregationWindowLength", signal.GetAggregationWindowLength())
		}
	}
}

func (p *segmentedPercentilePrediction) Name() string {
	return "SegmentedPercentile"
}
