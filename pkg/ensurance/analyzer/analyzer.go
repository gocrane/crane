package analyzer

import (
	"fmt"
	"sort"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
	"github.com/gocrane/crane/pkg/common"
	ecache "github.com/gocrane/crane/pkg/ensurance/cache"
	"github.com/gocrane/crane/pkg/ensurance/executor"
	einformer "github.com/gocrane/crane/pkg/ensurance/informer"
	"github.com/gocrane/crane/pkg/ensurance/logic"
	"github.com/gocrane/crane/pkg/ensurance/statestore"
	"github.com/gocrane/crane/pkg/utils"
	"github.com/gocrane/crane/pkg/utils/log"
)

type AnalyzerManager struct {
	nodeName          string
	podInformer       cache.SharedIndexInformer
	nodeInformer      cache.SharedIndexInformer
	nepInformer       cache.SharedIndexInformer
	avoidanceInformer cache.SharedIndexInformer
	statestore        statestore.StateStore
	recorder          record.EventRecorder
	noticeCh          chan<- executor.AvoidanceExecutor

	logic             logic.Logic
	status            map[string][]common.TimeSeries
	reached           map[string]uint64
	restored          map[string]uint64
	actionEventStatus map[string]ecache.DetectionStatus
	lastTriggeredTime time.Time
}

// AnalyzerManager create analyzer manager
func NewAnalyzerManager(nodeName string, podInformer cache.SharedIndexInformer, nodeInformer cache.SharedIndexInformer, nepInformer cache.SharedIndexInformer,
	avoidanceInformer cache.SharedIndexInformer, statestore statestore.StateStore, record record.EventRecorder, noticeCh chan<- executor.AvoidanceExecutor) Analyzer {

	basicLogic := logic.NewBasicLogic()

	return &AnalyzerManager{
		nodeName:          nodeName,
		logic:             basicLogic,
		noticeCh:          noticeCh,
		recorder:          record,
		podInformer:       podInformer,
		nodeInformer:      nodeInformer,
		nepInformer:       nepInformer,
		avoidanceInformer: avoidanceInformer,
		statestore:        statestore,
		reached:           make(map[string]uint64),
		restored:          make(map[string]uint64),
		actionEventStatus: make(map[string]ecache.DetectionStatus),
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
				s.Analyze()
			case <-stop:
				log.Logger().V(2).Info("Analyzer exit")
				return
			}
		}
	}()

	return
}

func (s *AnalyzerManager) Analyze() {
	// step1 copy neps
	node, err := einformer.GetNodeFromInformer(s.nodeInformer, s.nodeName)
	if err != nil {
		log.Logger().V(2).Info("Warning: get node name failed, not to do analyze")
		return
	}

	var neps []*ensuranceapi.NodeQOSEnsurancePolicy
	allNeps := s.nepInformer.GetStore().List()
	for _, n := range allNeps {
		nep := n.(*ensuranceapi.NodeQOSEnsurancePolicy)

		//check the node is selected by the
		if matched, err := utils.LabelSelectorMatched(node.Labels, &nep.Spec.LabelSelector); err != nil {
			log.Logger().V(2).Info(fmt.Sprintf("Warning: the nep label selector error,err: %s", err.Error()))
			continue
		} else if !matched {
			continue
		}

		neps = append(neps, nep.DeepCopy())
	}

	var avoidanceMaps = make(map[string]*ensuranceapi.AvoidanceAction)
	allAvoidance := s.avoidanceInformer.GetStore().List()
	for _, n := range allAvoidance {
		avoidance := n.(*ensuranceapi.AvoidanceAction)
		avoidanceMaps[avoidance.Name] = avoidance
	}

	s.status = s.statestore.List()

	// step 2: do analyze for neps
	var dcs []ecache.DetectionCondition
	for _, n := range neps {
		for _, v := range n.Spec.ObjectiveEnsurances {
			var key = strings.Join([]string{n.Name, v.Name}, ".")
			detection, err := s.doAnalyze(key, v)
			if err != nil {
				log.Logger().V(4).Info(fmt.Sprintf("Warning: doAnalyze failed %s", err.Error()))
			}
			detection.Nep = n
			dcs = append(dcs, detection)
		}
	}

	log.Logger().V(4).Info("Analyze:", "dcs", dcs)

	//step 3 : doMerge
	avoidanceAction := s.doMerge(avoidanceMaps, dcs)
	if err != nil {
		log.Logger().Error(err, "Analyze doMerge failed")
		return
	}

	//step 4 :notice the avoidance manager
	s.noticeAvoidanceManager(avoidanceAction)

	return
}

func (s *AnalyzerManager) doAnalyze(key string, object ensuranceapi.ObjectiveEnsurance) (ecache.DetectionCondition, error) {

	var dc = ecache.DetectionCondition{DryRun: object.DryRun, ObjectiveEnsuranceName: object.Name, ActionName: object.AvoidanceActionName}

	//step1: get metric value
	value, err := s.getMetricFromMap(object.MetricRule.Metric.Name, object.MetricRule.Metric.Selector)
	if err != nil {
		return dc, err
	}

	//step2: use opa to check if reached
	threshold, err := s.logic.EvalWithMetric(object.MetricRule.Metric.Name, float64(object.MetricRule.Target.Value.Value()), value)
	if err != nil {
		return dc, err
	}

	//step3: check is reached action or restored, set the detection
	if threshold {
		s.restored[key] = 0
		reached := utils.GetUint64FromMaps(key, s.reached)
		reached++
		s.reached[key] = reached
		if reached >= uint64(object.ReachedThreshold) {
			dc.Triggered = true
		}
	} else {
		s.reached[key] = 0
		restored := utils.GetUint64FromMaps(key, s.restored)
		restored++
		s.restored[key] = restored
		if restored >= uint64(object.RestoredThreshold) {
			dc.Restored = true
		}
	}

	return dc, nil
}

func (s *AnalyzerManager) doMerge(avoidanceMaps map[string]*ensuranceapi.AvoidanceAction, dcs []ecache.DetectionCondition) executor.AvoidanceExecutor {
	var now = time.Now()

	//step1 filter the only dryRun detection
	var dcsFiltered []ecache.DetectionCondition
	for _, dc := range dcs {
		s.doLogEvent(dc, now)
		if !dc.DryRun {
			dcsFiltered = append(dcsFiltered, dc)
		}
	}

	var ae executor.AvoidanceExecutor

	//step2 do DisableScheduled merge
	var disableScheduled bool
	var restoreScheduled bool
	for _, dc := range dcsFiltered {
		if dc.Triggered {
			disableScheduled = true
			restoreScheduled = false
		}
	}

	if disableScheduled {
		ae.ScheduledExecutor.DisableScheduledQOSPriority = &executor.ScheduledQOSPriority{PodQOSClass: v1.PodQOSBestEffort, PriorityClassValue: 0}
		s.lastTriggeredTime = now
	} else {
		if len(dcsFiltered) == 0 {
			restoreScheduled = true
		} else {
			for _, dc := range dcsFiltered {
				action, ok := avoidanceMaps[dc.ActionName]
				if !ok {
					log.Logger().V(4).Info(fmt.Sprintf("Waring: doMerge for detection the action %s  not found", dc.ActionName))
					continue
				}

				if dc.Restored {
					var schedulingCoolDown = utils.GetInt64withDefault(action.Spec.CoolDownSeconds, executor.DefaultCoolDownSeconds)
					log.Logger().V(4).Info("doMerge", "schedulingCoolDown", schedulingCoolDown)
					if now.After(s.lastTriggeredTime.Add(time.Duration(schedulingCoolDown) * time.Second)) {
						restoreScheduled = true
						break
					}
				}
			}
		}
	}

	if restoreScheduled {
		ae.ScheduledExecutor.RestoreScheduledQOSPriority = &executor.ScheduledQOSPriority{PodQOSClass: v1.PodQOSBestEffort, PriorityClassValue: 0}
	}

	//step3 do Throttle merge FilterAndSortThrottlePods
	//step4 do Evict merge  FilterAndSortEvictPods
	var basicEvictQosPriority = executor.ScheduledQOSPriority{PodQOSClass: v1.PodQOSBestEffort, PriorityClassValue: 0}
	var evictPods executor.EvictPods
	for _, dc := range dcsFiltered {
		if dc.Triggered {
			action, ok := avoidanceMaps[dc.ActionName]
			if !ok {
				log.Logger().V(4).Info("Waring: doMerge for detection the action ", dc.ActionName, " not found")
				continue
			}

			if action.Spec.Eviction != nil {
				var deletionGracePeriodSeconds = utils.GetInt32withDefault(action.Spec.Eviction.DeletionGracePeriodSeconds, executor.DefaultDeletionGracePeriodSeconds)
				var allPods = einformer.GetAllPodFromInformer(s.podInformer)
				for _, v := range allPods {
					var qosPriority = executor.ScheduledQOSPriority{PodQOSClass: v.Status.QOSClass, PriorityClassValue: utils.GetInt32withDefault(v.Spec.Priority, 0)}
					if !qosPriority.Greater(basicEvictQosPriority) {
						evictPods = append(evictPods, executor.EvictPod{DeletionGracePeriodSeconds: deletionGracePeriodSeconds,
							PodTypes: types.NamespacedName{Name: v.Name, Namespace: v.Namespace}, PodQOSPriority: qosPriority})
					}
				}
			}
		}
	}

	// remove the replicated pod
	for _, e := range evictPods {
		if i := ae.EvictExecutor.Executors.Find(e.PodTypes); i == -1 {
			ae.EvictExecutor.Executors = append(ae.EvictExecutor.Executors, e)
		} else {
			if e.DeletionGracePeriodSeconds < ae.EvictExecutor.Executors[i].DeletionGracePeriodSeconds {
				ae.EvictExecutor.Executors[i].DeletionGracePeriodSeconds = e.DeletionGracePeriodSeconds
			} else {
				// nothing to do,replicated filter the evictPod
			}
		}
	}

	// sort the evicting executor by pod qos priority
	sort.Sort(ae.EvictExecutor.Executors)

	return ae
}

func (s *AnalyzerManager) doLogEvent(dc ecache.DetectionCondition, now time.Time) {

	var key = strings.Join([]string{dc.Nep.Name, dc.ObjectiveEnsuranceName}, "/")

	if !(dc.Triggered || dc.Restored) {
		return
	}

	nodeRef := utils.GetNodeRef(s.nodeName)

	//step1 print log if the detection state is changed
	//step2 produce event
	if dc.Triggered {
		log.Logger().V(2).Info(fmt.Sprintf("%s triggered action %s", key, dc.ActionName))

		// record an event about the objective ensurance triggered
		s.recorder.Event(nodeRef, v1.EventTypeWarning, "ObjectiveEnsuranceTriggered", fmt.Sprintf("%s triggered action %s", key, dc.ActionName))
		s.actionEventStatus[key] = ecache.DetectionStatus{IsTriggered: true, LastTime: now}
	}

	if dc.Restored {
		if s.needSendEventForRestore(dc) {
			log.Logger().V(2).Info(fmt.Sprintf("%s restored action %s", key, dc.ActionName))
			// record an event about the objective ensurance restored
			s.recorder.Event(nodeRef, v1.EventTypeNormal, "ObjectiveEnsuranceRestored", fmt.Sprintf("%s restored action %s", key, dc.ActionName))
			s.actionEventStatus[key] = ecache.DetectionStatus{IsTriggered: false, LastTime: now}
		}
	}

	return
}

func (s *AnalyzerManager) getMetricFromMap(metricName string, selector *metav1.LabelSelector) (float64, error) {
	// step1: get the value from map
	if v, ok := s.status[metricName]; ok {
		for _, vv := range v {
			if matched, err := utils.LabelSelectorMatched(common.Labels2Maps(vv.Labels), selector); err != nil {
				return 0.0, err
			} else if !matched {
				continue
			} else {
				return vv.Samples[0].Value, nil
			}
		}
	}

	return 0.0, fmt.Errorf("metricName %s not found value", metricName)
}

func (s *AnalyzerManager) noticeAvoidanceManager(as executor.AvoidanceExecutor) {
	//step1: check need to notice avoidance manager

	//step2: notice by channel
	s.noticeCh <- as
	return
}

func (s *AnalyzerManager) needSendEventForRestore(dc ecache.DetectionCondition) bool {
	var key = strings.Join([]string{dc.Nep.Name, dc.ObjectiveEnsuranceName}, "/")

	if v, ok := s.actionEventStatus[key]; ok {
		if dc.Restored {
			if v.IsTriggered {
				return true
			}
		}
	}

	return false
}
