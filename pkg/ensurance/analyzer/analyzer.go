package analyzer

import (
	"github.com/gocrane/crane/pkg/ensurance/logic"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
	ecache "github.com/gocrane/crane/pkg/ensurance/cache"
	"github.com/gocrane/crane/pkg/ensurance/executor"
	"github.com/gocrane/crane/pkg/utils/clogs"
)

type AnalyzerManager struct {
	podInformer       cache.SharedIndexInformer
	nodeInformer      cache.SharedIndexInformer
	nepInformer       cache.SharedIndexInformer
	avoidanceInformer cache.SharedIndexInformer
	noticeCh          chan<- executor.AvoidanceExecutorStruct

	logic      logic.Logic
	NodeStatus sync.Map
	dcsOlder   []ecache.DetectionCondition
	acsOlder   executor.AvoidanceExecutorStruct
}

// AnalyzerManager create analyzer manager
func NewAnalyzerManager(podInformer cache.SharedIndexInformer, nodeInformer cache.SharedIndexInformer, nepInformer cache.SharedIndexInformer,
	avoidanceInformer cache.SharedIndexInformer, noticeCh chan<- executor.AvoidanceExecutorStruct) Analyzer {

	opaLogic := logic.NewOpaLogic()

	return &AnalyzerManager{
		logic:             opaLogic,
		noticeCh:          noticeCh,
		podInformer:       podInformer,
		nodeInformer:      nodeInformer,
		nepInformer:       nepInformer,
		avoidanceInformer: avoidanceInformer,
	}
}

func (s *AnalyzerManager) Name() string {
	return "AnalyzeManager"
}

func (s *AnalyzerManager) Run(stop <-chan struct{}) {
	go func() {
		updateTicker := time.NewTicker(10 * time.Second)
		defer updateTicker.Stop()
		for {
			select {
			case <-updateTicker.C:
				clogs.Log().V(2).Info("Analyzer run periodically")
				s.Analyze()
			case <-stop:
				clogs.Log().V(2).Info("Analyzer exit")
				return
			}
		}
	}()

	return
}

func (s *AnalyzerManager) Analyze() {
	// step1 copy neps
	var neps []*ensuranceapi.NodeQOSEnsurancePolicy
	allNeps := s.nepInformer.GetStore().List()
	for _, n := range allNeps {
		nep := n.(*ensuranceapi.NodeQOSEnsurancePolicy)
		neps = append(neps, nep.DeepCopy())
	}

	// step 2: do analyze for neps
	var dcs []ecache.DetectionCondition
	for _, n := range neps {
		for _, v := range n.Spec.ObjectiveEnsurance {
			detection, err := s.doAnalyze(v)
			if err != nil {
				//warning and continue
			}
			detection.PolicyName = n.Name
			detection.Namespace = n.Namespace
			detection.ObjectiveEnsuranceName = v.AvoidanceActionName
			dcs = append(dcs, detection)
		}
	}

	//step 3: log and event
	s.doLogEvent(dcs)

	//step 4 : doMerge
	avoidanceActionStruct, err := s.doMerge(dcs)
	if err != nil {
		// to return err
	}

	//step 5 :notice the avoidance manager
	s.noticeAvoidanceManager(avoidanceActionStruct)

	return
}

func (s *AnalyzerManager) doAnalyze(object ensuranceapi.ObjectiveEnsurance) (ecache.DetectionCondition, error) {
	//step1: get metric value
	value, err := s.getMetricFromMap(object.MetricRule.Metric.Name, object.MetricRule.Metric.Selector)
	if err != nil {
		return ecache.DetectionCondition{}, err
	}

	//step2: use opa to check if reached
	s.logic.EvalWithMetric(object.MetricRule.Metric.Name, float64(object.MetricRule.Target.Value.Value()), value)

	//step3: check is reached action or restored, set the detection

	return ecache.DetectionCondition{}, nil
}

func (s *AnalyzerManager) doMerge(dcs []ecache.DetectionCondition) (executor.AvoidanceExecutorStruct, error) {
	//step1 filter the only dryRun detection
	//step2 do BlockScheduled merge
	//step3 do Throttle merge FilterAndSortThrottlePods
	//step4 do Evict merge  FilterAndSortEvictPods
	return executor.AvoidanceExecutorStruct{}, nil
}

func (a *AnalyzerManager) doLogEvent(dcs []ecache.DetectionCondition) {
	//step1 print log if the detection state is changed
	//step2 produce event
}

func (s *AnalyzerManager) getMetricFromMap(metricName string, selector *metav1.LabelSelector) (float64, error) {
	// step1: generate the key for the metric
	// step2: get the value from map
	return 0.0, nil
}

func (s *AnalyzerManager) noticeAvoidanceManager(as executor.AvoidanceExecutorStruct) {
	//step1: check need to notice avoidance manager

	//step2: notice by channel
	s.noticeCh <- as
	return
}
