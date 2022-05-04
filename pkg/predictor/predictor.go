package predictor

import (
	"sync"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"

	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/checkpoint"
	"github.com/gocrane/crane/pkg/prediction"
	predconf "github.com/gocrane/crane/pkg/prediction/config"
	"github.com/gocrane/crane/pkg/prediction/dsp"
	"github.com/gocrane/crane/pkg/prediction/percentile"
	"github.com/gocrane/crane/pkg/providers"
)

type AlgorithmDataProviders struct {
	// RealTimeProviders is names of realtime data providers. now support `prom`, `metricserver`
	RealTimeProviders []providers.DataSourceType
	// HistoryProviders is names of history data providers. now support only `prom`.
	HistoryProviders []providers.DataSourceType
}

type Config struct {
	DataProviders AlgorithmDataProviders
	ModelConfig   predconf.AlgorithmModelConfig
}

// DefaultPredictorsConfig will use all datasources you for real time and history provider. data proxy will select the first available.
// Now, for RealTimeProvider if you specified metricserver in command args, it is [metricserver,prom] in order, if not, it is [prom]. for HistoryProvider is [prom]
func DefaultPredictorsConfig(modelConfig predconf.AlgorithmModelConfig) map[predictionapi.AlgorithmType]Config {
	configs := map[predictionapi.AlgorithmType]Config{
		predictionapi.AlgorithmTypeDSP: {
			ModelConfig: modelConfig,
		},
		predictionapi.AlgorithmTypePercentile: {
			DataProviders: AlgorithmDataProviders{},
		},
	}
	return configs
}

type Manager interface {
	// Start start all predictors in manager, block until stopCh and all predictors exited
	Start(stopCh <-chan struct{})
	// GetPredictor return a registered predictor
	GetPredictor(predictor predictionapi.AlgorithmType) prediction.Interface
	// AddPredictorRealTimeProvider Dynamically add a real time data provider for a predictor
	AddPredictorRealTimeProvider(predictorName predictionapi.AlgorithmType, dataProviderName providers.DataSourceType, dataProvider providers.RealTime)
	// DeletePredictorRealTimeProvider Dynamically delete a real time data provider for a predictor
	DeletePredictorRealTimeProvider(predictorName predictionapi.AlgorithmType, dataProviderName providers.DataSourceType)
	// AddPredictorHistoryProvider Dynamically add a history data provider for a predictor
	AddPredictorHistoryProvider(predictorName predictionapi.AlgorithmType, dataProviderName providers.DataSourceType, dataProvider providers.History)
	// DeletePredictorHistoryProvider Dynamically delete a history data provider for a predictor
	DeletePredictorHistoryProvider(predictorName predictionapi.AlgorithmType, dataProviderName providers.DataSourceType)
}

type manager struct {
	lock       sync.Mutex
	predictors map[predictionapi.AlgorithmType]prediction.Interface
	// different type of data proxy may share the same underlying data source. But when you use a remote predictor, then this data source is not needed
	// each algorithm predictor has its own real time data proxy
	realTimeDataProxys map[predictionapi.AlgorithmType]*providers.RealTimeDataProxy
	// each algorithm predictor has its own history data proxy
	historyDataProxys map[predictionapi.AlgorithmType]*providers.HistoryDataProxy
}

type CheckPointerContext struct {
	Enable       bool
	Checkpointer checkpoint.Checkpointer
}

func NewManager(realtimeProviders map[providers.DataSourceType]providers.RealTime,
	historyProviders map[providers.DataSourceType]providers.History, predictorsConfig map[predictionapi.AlgorithmType]Config,
	checkpointerCtx CheckPointerContext) Manager {

	m := &manager{
		predictors:         make(map[predictionapi.AlgorithmType]prediction.Interface),
		realTimeDataProxys: make(map[predictionapi.AlgorithmType]*providers.RealTimeDataProxy),
		historyDataProxys:  make(map[predictionapi.AlgorithmType]*providers.HistoryDataProxy),
	}
	for algo, predictorConf := range predictorsConfig {
		var algorithmRealTimeProxy *providers.RealTimeDataProxy
		var algorithmHistoryProxy *providers.HistoryDataProxy
		// Default use all realtime providers if predictorConf not specified the algorithm real time data providers
		if len(predictorConf.DataProviders.HistoryProviders) == 0 {
			algorithmHistoryProxy = providers.NewHistoryDataProxy(historyProviders)
		} else {
			algoHistProviders := make(map[providers.DataSourceType]providers.History)
			for _, histProviderName := range predictorConf.DataProviders.HistoryProviders {
				if histProvider, ok := historyProviders[histProviderName]; ok {
					algoHistProviders[histProviderName] = histProvider
				}
			}
			algorithmHistoryProxy = providers.NewHistoryDataProxy(algoHistProviders)
		}
		// Default use all history providers if predictorConf not specified the algorithm history data providers
		if len(predictorConf.DataProviders.RealTimeProviders) == 0 {
			algorithmRealTimeProxy = providers.NewRealTimeDataProxy(realtimeProviders)
		} else {
			algoRealTimeProviders := make(map[providers.DataSourceType]providers.RealTime)
			for _, realtimeProviderName := range predictorConf.DataProviders.RealTimeProviders {
				if realtimeProvider, ok := realtimeProviders[realtimeProviderName]; ok {
					algoRealTimeProviders[realtimeProviderName] = realtimeProvider
				}
			}
			algorithmRealTimeProxy = providers.NewRealTimeDataProxy(algoRealTimeProviders)
		}

		switch algo {
		case predictionapi.AlgorithmTypePercentile:
			pctPredictor := percentile.NewPrediction(algorithmRealTimeProxy, algorithmHistoryProxy, checkpointerCtx.Enable, checkpointerCtx.Checkpointer)
			m.predictors[algo] = pctPredictor
			m.historyDataProxys[algo] = algorithmHistoryProxy
			m.realTimeDataProxys[algo] = algorithmRealTimeProxy
		case predictionapi.AlgorithmTypeDSP:
			dspPredictor := dsp.NewPrediction(algorithmRealTimeProxy, algorithmHistoryProxy, predictorConf.ModelConfig)
			m.predictors[algo] = dspPredictor
			m.historyDataProxys[algo] = algorithmHistoryProxy
			m.realTimeDataProxys[algo] = algorithmRealTimeProxy
		default:
			klog.Errorf("Unknown predictor %v", algo)
			continue
		}
	}
	klog.Infof("predictors %+v", m.predictors)
	return m
}

func (m *manager) Start(stopCh <-chan struct{}) {

	var wg sync.WaitGroup
	func() {
		m.lock.Lock()
		defer m.lock.Unlock()
		for _, predictor := range m.predictors {
			wg.Add(1)
			go func(predictor prediction.Interface) {
				defer utilruntime.HandleCrash()
				defer wg.Done()
				predictor.Run(stopCh)
			}(predictor)
		}
	}()

	klog.Infof("predictor manager started, all predictors started")

	wg.Wait()

	klog.Infof("predictor manager stopped, all predictors exited")
}

func (m *manager) GetPredictor(predictor predictionapi.AlgorithmType) prediction.Interface {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.predictors[predictor]
}

// AddPredictorRealTimeProvider adds a real time data provider for a predictor
func (m *manager) AddPredictorRealTimeProvider(predictorName predictionapi.AlgorithmType, dataProviderName providers.DataSourceType, dataProvider providers.RealTime) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if proxy, ok := m.realTimeDataProxys[predictorName]; ok && proxy != nil {
		proxy.RegisterRealTimeProvider(dataProviderName, dataProvider)
	}
}

// DeletePredictorRealTimeProvider deletes a real time data provider
func (m *manager) DeletePredictorRealTimeProvider(predictorName predictionapi.AlgorithmType, dataProviderName providers.DataSourceType) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if proxy, ok := m.realTimeDataProxys[predictorName]; ok && proxy != nil {
		proxy.DeleteRealTimeProvider(dataProviderName)
	}
}

// AddPredictorHistoryProvider adds a history data provider for a predictor
func (m *manager) AddPredictorHistoryProvider(predictorName predictionapi.AlgorithmType, dataProviderName providers.DataSourceType, dataProvider providers.History) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if proxy, ok := m.historyDataProxys[predictorName]; ok && proxy != nil {
		proxy.RegisterHistoryProvider(dataProviderName, dataProvider)
	}
}

// DeletePredictorHistoryProvider deletes the history data provider
func (m *manager) DeletePredictorHistoryProvider(predictorName predictionapi.AlgorithmType, dataProviderName providers.DataSourceType) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if proxy, ok := m.historyDataProxys[predictorName]; ok && proxy != nil {
		proxy.DeleteHistoryProvider(dataProviderName)
	}
}
