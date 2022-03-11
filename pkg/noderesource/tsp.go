package noderesource

import (
	"fmt"
	predictionv1alpha1 "github.com/gocrane/api/pkg/generated/informers/externalversions/prediction/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/prediction/config"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"strconv"
)

const (
	TspNamespace                      = "default"
	timeSeriesPredictionCollectorName = "TimeSeriesPrediction"
)

func init() {
	klog.Infof("init TimeSeriesPredictionCollector")
	registerMetrics(timeSeriesPredictionCollectorName, NewTimeSeriesPredictionCollector)
}

type TimeSeriesPredictionCollector struct {
	timeSeriesPredictionInformer predictionv1alpha1.TimeSeriesPredictionInformer
	recorder                     record.EventRecorder
	nodeName                     string
	tspName                      string
}

func NewTimeSeriesPredictionCollector(context *CollectContext) (Collector, error) {
	klog.V(4).Infof("create TimeSeriesPredictionCollector")
	return &TimeSeriesPredictionCollector{
		timeSeriesPredictionInformer: context.TimeSeriesPredictionInformer,
		recorder:                     context.Recorder,
		nodeName:                     context.NodeName,
		tspName:                      context.TspName,
	}, nil
}

func (r *TimeSeriesPredictionCollector) Run(stop <-chan struct{}, stateChan chan struct {
	stateMap      map[string][]MetricTimeSeries
	collectorName string
}) {
	klog.Infof("Add timeSeriesPredictionInformer event handler")
	r.timeSeriesPredictionInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			klog.Infof("tsp update")
			tsp, ok := newObj.(*predictionapi.TimeSeriesPrediction)
			if ok {
				go r.Reconcile(tsp, stateChan)
			}
		},
		AddFunc: func(obj interface{}) {
			klog.Infof("tsp create")
			tsp, ok := obj.(*predictionapi.TimeSeriesPrediction)
			if ok {
				go r.Reconcile(tsp, stateChan)
			}
		},
	})
	r.timeSeriesPredictionInformer.Informer()
}

func (r *TimeSeriesPredictionCollector) Reconcile(tsp *predictionapi.TimeSeriesPrediction, stateChan chan struct {
	stateMap      map[string][]MetricTimeSeries
	collectorName string
}) {
	klog.Infof("Node resource reconcile: %s/%s", tsp.Namespace, tsp.Name)
	// get current node info
	target := tsp.Spec.TargetRef
	if target.Kind != config.TargetKindNode || target.Name != r.nodeName {
		return
	}
	if tsp.Name != r.tspName {
		return
	}

	stateChan <- struct {
		stateMap      map[string][]MetricTimeSeries
		collectorName string
	}{stateMap: r.collect(tsp), collectorName: r.Name()}

	return
}

func (r *TimeSeriesPredictionCollector) GetLastState() map[string][]MetricTimeSeries {
	result := make(map[string][]MetricTimeSeries)
	tsp, err := r.timeSeriesPredictionInformer.Lister().TimeSeriesPredictions(TspNamespace).Get(r.tspName)
	if err != nil {
		return result
	}
	return r.collect(tsp)
}

func (r *TimeSeriesPredictionCollector) collect(tsp *predictionapi.TimeSeriesPrediction) map[string][]MetricTimeSeries {
	klog.V(4).Infof("tsp start collect")
	stateMap := make(map[string][]MetricTimeSeries, 0)
	idToResourceMap := map[string]v1.ResourceName{
		v1.ResourceCPU.String():    v1.ResourceCPU,
		v1.ResourceMemory.String(): v1.ResourceMemory,
	}

	if tsp == nil {
		return stateMap
	}

	// build node status
	nextPredictionResourceStatus := &tsp.Status
	for _, metrics := range nextPredictionResourceStatus.PredictionMetrics {
		resourceName, exists := idToResourceMap[metrics.ResourceIdentifier]
		if !exists {
			continue
		}

		var metricTimeSeriesList []MetricTimeSeries
		if metricTimeSeriesList, exists = stateMap[resourceName.String()]; !exists {
			metricTimeSeriesList = make([]MetricTimeSeries, 0)
		}

		metricTimeSeriesList = append(metricTimeSeriesList, MetricTimeSeries{
			DataSourceName: fmt.Sprintf("tsp:%s:%s", tsp.Name, metrics.ResourceIdentifier),
			TimeSeriesList: ApiTimeSeriesToCommonTimeSeries(metrics.Prediction),
		})
		stateMap[resourceName.String()] = metricTimeSeriesList
	}
	return stateMap
}

func (r *TimeSeriesPredictionCollector) Name() string {
	return timeSeriesPredictionCollectorName
}

func ApiTimeSeriesToCommonTimeSeries(series []*predictionapi.MetricTimeSeries) []common.TimeSeries {
	var list = make([]common.TimeSeries, 0)
	for _, mts := range series {
		ts := common.TimeSeries{
			Labels:  make([]common.Label, len(mts.Labels)),
			Samples: make([]common.Sample, len(mts.Samples)),
		}
		for i, label := range mts.Labels {
			ts.Labels[i] = common.Label{Name: label.Name, Value: label.Value}
		}
		for i, sample := range mts.Samples {
			v, _ := strconv.ParseFloat(sample.Value, 64)
			ts.Samples[i] = common.Sample{Timestamp: sample.Timestamp, Value: v}
		}
		list = append(list, ts)
	}
	return list
}
