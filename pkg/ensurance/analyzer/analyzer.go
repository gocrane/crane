package analyzer

import (
	"fmt"
	"sort"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"

	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
	"github.com/gocrane/api/pkg/generated/informers/externalversions/ensurance/v1alpha1"
	ensurancelisters "github.com/gocrane/api/pkg/generated/listers/ensurance/v1alpha1"
	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/analyzer/evaluator"
	ecache "github.com/gocrane/crane/pkg/ensurance/cache"
	stypes "github.com/gocrane/crane/pkg/ensurance/collector/types"
	"github.com/gocrane/crane/pkg/ensurance/executor"
	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/utils"
)

type AnormalyAnalyzer struct {
	nodeName string

	podLister corelisters.PodLister
	podSynced cache.InformerSynced

	nodeLister corelisters.NodeLister
	nodeSynced cache.InformerSynced

	nodeQOSLister ensurancelisters.NodeQOSEnsurancePolicyLister
	nodeQOSSynced cache.InformerSynced

	avoidanceActionLister ensurancelisters.AvoidanceActionLister
	avoidanceActionSynced cache.InformerSynced

	stateChann chan map[string][]common.TimeSeries
	recorder   record.EventRecorder
	actionCh   chan<- executor.AvoidanceExecutor

	evaluator         evaluator.Evaluator
	triggered         map[string]uint64
	restored          map[string]uint64
	actionEventStatus map[string]ecache.DetectionStatus
	lastTriggeredTime time.Time
}

// NewAnormalyAnalyzer create an analyzer manager
func NewAnormalyAnalyzer(kubeClient *kubernetes.Clientset,
	nodeName string,
	podInformer coreinformers.PodInformer,
	nodeInformer coreinformers.NodeInformer,
	nepInformer v1alpha1.NodeQOSEnsurancePolicyInformer,
	actionInformer v1alpha1.AvoidanceActionInformer,
	stateChann chan map[string][]common.TimeSeries,
	noticeCh chan<- executor.AvoidanceExecutor,
) *AnormalyAnalyzer {

	expressionEvaluator := evaluator.NewExpressionEvaluator()
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "crane-agent"})
	return &AnormalyAnalyzer{
		nodeName:              nodeName,
		evaluator:             expressionEvaluator,
		actionCh:              noticeCh,
		recorder:              recorder,
		podLister:             podInformer.Lister(),
		podSynced:             podInformer.Informer().HasSynced,
		nodeLister:            nodeInformer.Lister(),
		nodeSynced:            nodeInformer.Informer().HasSynced,
		nodeQOSLister:         nepInformer.Lister(),
		nodeQOSSynced:         nepInformer.Informer().HasSynced,
		avoidanceActionLister: actionInformer.Lister(),
		avoidanceActionSynced: actionInformer.Informer().HasSynced,
		stateChann:            stateChann,
		triggered:             make(map[string]uint64),
		restored:              make(map[string]uint64),
		actionEventStatus:     make(map[string]ecache.DetectionStatus),
	}
}

func (s *AnormalyAnalyzer) Name() string {
	return "AnalyzeManager"
}

func (s *AnormalyAnalyzer) Run(stop <-chan struct{}) {
	klog.Infof("Starting anormaly analyzer.")

	// Wait for the caches to be synced before starting workers
	if !cache.WaitForNamedCacheSync("anormaly-analyzer",
		stop,
		s.podSynced,
		s.nodeSynced,
		s.nodeQOSSynced,
		s.avoidanceActionSynced,
	) {
		return
	}

	go func() {
		for {
			select {
			case state := <-s.stateChann:
				go s.Analyze(state)
			case <-stop:
				klog.Infof("AnormalyAnalyzer exit")
				return
			}
		}
	}()

	return
}

func (s *AnormalyAnalyzer) Analyze(state map[string][]common.TimeSeries) {
	node, err := s.nodeLister.Get(s.nodeName)
	if err != nil {
		klog.Errorf("Failed to get node: %v", err)
		return
	}

	var neps []*ensuranceapi.NodeQOSEnsurancePolicy
	allNeps, err := s.nodeQOSLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list NodeQOS: %v", err)
		return
	}

	for _, nep := range allNeps {
		if matched, err := utils.LabelSelectorMatched(node.Labels, nep.Spec.Selector); err != nil || !matched {
			continue
		}
		neps = append(neps, nep.DeepCopy())
	}

	var avoidanceMaps = make(map[string]*ensuranceapi.AvoidanceAction)
	allAvoidance, err := s.avoidanceActionLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list AvoidanceActions, %v", err)
		return
	}

	for _, a := range allAvoidance {
		avoidanceMaps[a.Name] = a
	}

	// step 2: do analyze for neps
	var actionContexts []ecache.ActionContext
	for _, n := range neps {
		for _, v := range n.Spec.ObjectiveEnsurances {
			var key = strings.Join([]string{n.Name, v.Name}, ".")
			ac, err := s.analyze(key, v, state)
			if err != nil {
				klog.Errorf("Failed to analyze, %v.", err)
			}
			ac.Nep = n
			actionContexts = append(actionContexts, ac)
		}
	}

	klog.V(6).Infof("Analyze actionContexts: %v", actionContexts)

	//step 3 : merge
	avoidanceAction := s.merge(state, avoidanceMaps, actionContexts)
	if err != nil {
		klog.Errorf("Failed to merge, %v.", err)
		return
	}

	//step 4 :notice the enforcer manager
	s.notify(avoidanceAction)

	return
}

func (s *AnormalyAnalyzer) getSeries(state []common.TimeSeries, selector *metav1.LabelSelector, metricName string) ([]common.TimeSeries, error) {
	series := s.getTimeSeriesFromMap(state, selector)
	if len(series) == 0 {
		return []common.TimeSeries{}, fmt.Errorf("time series length is 0 for metric %s", metricName)
	}
	return series, nil
}

func (s *AnormalyAnalyzer) trigger(series []common.TimeSeries, object ensuranceapi.ObjectiveEnsurance) bool {
	var triggered, threshold bool
	for _, ts := range series {
		triggered = s.evaluator.EvalWithMetric(object.MetricRule.Name, float64(object.MetricRule.Value.Value()), ts.Samples[0].Value)

		klog.V(6).Infof("Anormaly detection result %v, Name: %s, Value: %.2f, %s/%s", triggered,
			object.MetricRule.Name,
			ts.Samples[0].Value,
			common.GetValueByName(ts.Labels, common.LabelNamePodNamespace),
			common.GetValueByName(ts.Labels, common.LabelNamePodName))

		if triggered {
			threshold = true
		}
	}
	return threshold
}

func (s *AnormalyAnalyzer) analyze(key string, object ensuranceapi.ObjectiveEnsurance, stateMap map[string][]common.TimeSeries) (ecache.ActionContext, error) {
	var ac = ecache.ActionContext{Strategy: object.Strategy, ObjectiveEnsuranceName: object.Name, ActionName: object.AvoidanceActionName}

	state, ok := stateMap[object.MetricRule.Name]
	if !ok {
		return ac, fmt.Errorf("metric %s not found", object.MetricRule.Name)
	}

	//step1: get series from value
	series, err := s.getSeries(state, object.MetricRule.Selector, object.MetricRule.Name)
	if err != nil {
		return ac, err
	}

	//step2: check if triggered for NodeQOSEnsurance
	threshold := s.trigger(series, object)

	klog.V(4).Infof("for NodeQOS %s, metrics reach the threshold: %v", key, threshold)

	//step3: check is triggered action or restored, set the detection
	s.computeActionContext(threshold, key, object, &ac)

	return ac, nil
}

func (s *AnormalyAnalyzer) computeActionContext(threshold bool, key string, object ensuranceapi.ObjectiveEnsurance, ac *ecache.ActionContext) {
	if threshold {
		s.restored[key] = 0
		triggered := utils.GetUint64FromMaps(key, s.triggered)
		triggered++
		s.triggered[key] = triggered
		if triggered >= uint64(utils.GetInt32withDefault(object.AvoidanceThreshold, known.DefaultAvoidedThreshold)) {
			ac.Triggered = true
		}
	} else {
		s.triggered[key] = 0
		restored := utils.GetUint64FromMaps(key, s.restored)
		restored++
		s.restored[key] = restored
		if restored >= uint64(utils.GetInt32withDefault(object.RestoreThreshold, known.DefaultRestoredThreshold)) {
			ac.Restored = true
		}
	}
}

func (s *AnormalyAnalyzer) filterDryRun(acs []ecache.ActionContext) []ecache.ActionContext {
	var dcsFiltered []ecache.ActionContext
	now := time.Now()
	for _, ac := range acs {
		s.logEvent(ac, now)
		if !(ac.Strategy == ensuranceapi.AvoidanceActionStrategyPreview) {
			dcsFiltered = append(dcsFiltered, ac)
		}
	}
	return dcsFiltered
}

func (s *AnormalyAnalyzer) merge(stateMap map[string][]common.TimeSeries, avoidanceMaps map[string]*ensuranceapi.AvoidanceAction, acs []ecache.ActionContext) executor.AvoidanceExecutor {
	var ae executor.AvoidanceExecutor

	//step1 filter dry run ActionContext
	acsFiltered := s.filterDryRun(acs)

	//step2 do DisableScheduled merge
	enableSchedule := s.disableSchedulingMerge(acsFiltered, avoidanceMaps, &ae)

	for _, ac := range acsFiltered {
		action, ok := avoidanceMaps[ac.ActionName]
		if !ok {
			klog.Warningf("The action %s not found.", ac.ActionName)
			continue
		}

		//step3 get and deduplicate throttlePods, throttleUpPods
		if action.Spec.Throttle != nil {
			throttlePods, throttleUpPods := s.getThrottlePods(enableSchedule, ac, action, stateMap)
			// combine the replicated pod
			combineThrottleDuplicate(&ae.ThrottleExecutor, throttlePods, throttleUpPods)
		}

		//step4 get and deduplicate evictPods
		if action.Spec.Eviction != nil {
			evictPods := s.getEvictPods(ac.Triggered, action)
			// combine the replicated pod
			combineEvictDuplicate(&ae.EvictExecutor, evictPods)
		}
	}

	// sort the throttle executor by pod qos priority
	sort.Sort(ae.ThrottleExecutor.ThrottleDownPods)
	sort.Sort(sort.Reverse(ae.ThrottleExecutor.ThrottleUpPods))

	// sort the evict executor by pod qos priority
	sort.Sort(ae.EvictExecutor.EvictPods)

	return ae
}

func (s *AnormalyAnalyzer) logEvent(ac ecache.ActionContext, now time.Time) {
	var key = strings.Join([]string{ac.Nep.Name, ac.ObjectiveEnsuranceName}, "/")

	if !(ac.Triggered || ac.Restored) {
		return
	}

	nodeRef := utils.GetNodeRef(s.nodeName)

	//step1 print log if the detection state is changed
	//step2 produce event
	if ac.Triggered {
		klog.V(4).Infof("LOG: %s triggered action %s", key, ac.ActionName)

		// record an event about the objective ensurance triggered
		s.recorder.Event(nodeRef, v1.EventTypeWarning, "AvoidanceTriggered", fmt.Sprintf("%s triggered action %s", key, ac.ActionName))
		s.actionEventStatus[key] = ecache.DetectionStatus{IsTriggered: true, LastTime: now}
	}

	if ac.Restored {
		if s.actionTriggered(ac) {
			klog.V(4).Infof("LOG: %s restored action %s", key, ac.ActionName)
			// record an event about the objective ensurance restored
			s.recorder.Event(nodeRef, v1.EventTypeNormal, "RestoreTriggered", fmt.Sprintf("%s restored action %s", key, ac.ActionName))
			s.actionEventStatus[key] = ecache.DetectionStatus{IsTriggered: false, LastTime: now}
		}
	}

	return
}

func (s *AnormalyAnalyzer) getTimeSeriesFromMap(state []common.TimeSeries, selector *metav1.LabelSelector) []common.TimeSeries {
	var series []common.TimeSeries

	// step1: get the series from maps
	for _, vv := range state {
		if matched, err := utils.LabelSelectorMatched(common.Labels2Maps(vv.Labels), selector); err != nil {
			continue
		} else if !matched {
			continue
		} else {
			series = append(series, vv)
		}
	}
	return series
}

func (s *AnormalyAnalyzer) notify(as executor.AvoidanceExecutor) {
	//step1: check need to notice enforcer manager

	//step2: notice by channel
	s.actionCh <- as
	return
}

func (s *AnormalyAnalyzer) actionTriggered(ac ecache.ActionContext) bool {
	var key = strings.Join([]string{ac.Nep.Name, ac.ObjectiveEnsuranceName}, "/")

	if v, ok := s.actionEventStatus[key]; ok {
		if ac.Restored {
			if v.IsTriggered {
				return true
			}
		}
	}

	return false
}

func (s *AnormalyAnalyzer) getThrottlePods(enableSchedule bool, ac ecache.ActionContext,
	action *ensuranceapi.AvoidanceAction, stateMap map[string][]common.TimeSeries) ([]executor.ThrottlePod, []executor.ThrottlePod) {

	throttlePods, throttleUpPods := []executor.ThrottlePod{}, []executor.ThrottlePod{}

	allPods, err := s.podLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list all pods: %v", err)
		return throttlePods, throttleUpPods
	}

	for _, pod := range allPods {
		if ac.Triggered {
			throttlePods = append(throttlePods, throttlePodConstruct(pod, stateMap, action))
		}
		if enableSchedule && ac.Restored {
			throttleUpPods = append(throttleUpPods, throttlePodConstruct(pod, stateMap, action))
		}
	}

	return throttlePods, throttleUpPods
}

func (s *AnormalyAnalyzer) getEvictPods(triggered bool, action *ensuranceapi.AvoidanceAction) []executor.EvictPod {
	evictPods := []executor.EvictPod{}

	if triggered {
		allPods, err := s.podLister.List(labels.Everything())
		if err != nil {
			klog.Errorf("Failed to list all pods: %v.", err)
			return evictPods
		}

		for _, v := range allPods {
			var classAndPriority = executor.ClassAndPriority{PodQOSClass: v.Status.QOSClass, PriorityClassValue: utils.GetInt32withDefault(v.Spec.Priority, 0)}
			evictPods = append(evictPods, executor.EvictPod{DeletionGracePeriodSeconds: uint32(utils.GetInt32withDefault(action.Spec.Eviction.TerminationGracePeriodSeconds, known.DefaultDeletionGracePeriodSeconds)),
				PodKey: types.NamespacedName{Name: v.Name, Namespace: v.Namespace}, ClassAndPriority: classAndPriority})
		}
	}
	return evictPods
}

func (s *AnormalyAnalyzer) disableSchedulingMerge(acsFiltered []ecache.ActionContext, avoidanceMaps map[string]*ensuranceapi.AvoidanceAction, ae *executor.AvoidanceExecutor) (enableSchedule bool) {
	var now = time.Now()
	enableSchedule = false
	for _, ac := range acsFiltered {
		if ac.Triggered {
			enableSchedule = false
		}
		if ac.Restored {
			action, ok := avoidanceMaps[ac.ActionName]
			if !ok {
				klog.Warningf("DoMerge for detection,but the action %s not found", ac.ActionName)
				continue
			}
			var schedulingCoolDown = utils.GetInt32withDefault(action.Spec.CoolDownSeconds, executor.DefaultCoolDownSeconds)
			if !enableSchedule && now.After(s.lastTriggeredTime.Add(time.Duration(schedulingCoolDown)*time.Second)) {
				enableSchedule = true
				return
			}
		}
	}

	if enableSchedule {
		ae.ScheduleExecutor.RestoreClassAndPriority = &executor.ClassAndPriority{PodQOSClass: v1.PodQOSBestEffort, PriorityClassValue: 0}
	} else {
		ae.ScheduleExecutor.DisableClassAndPriority = &executor.ClassAndPriority{PodQOSClass: v1.PodQOSBestEffort, PriorityClassValue: 0}
	}
	s.lastTriggeredTime = now
	return
}

func throttlePodConstruct(pod *v1.Pod, stateMap map[string][]common.TimeSeries, action *ensuranceapi.AvoidanceAction) executor.ThrottlePod {
	var throttlePod executor.ThrottlePod
	var qosPriority = executor.ClassAndPriority{PodQOSClass: pod.Status.QOSClass, PriorityClassValue: utils.GetInt32withDefault(pod.Spec.Priority, 0)}

	throttlePod.PodTypes = types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}
	throttlePod.CPUThrottle.MinCPURatio = uint64(action.Spec.Throttle.CPUThrottle.MinCPURatio)
	throttlePod.CPUThrottle.StepCPURatio = uint64(action.Spec.Throttle.CPUThrottle.StepCPURatio)

	throttlePod.PodCPUUsage, throttlePod.ContainerCPUUsages = executor.GetPodUsage(string(stypes.MetricNameContainerCpuTotalUsage), stateMap, pod)
	throttlePod.PodCPUShare, throttlePod.ContainerCPUShares = executor.GetPodUsage(string(stypes.MetricNameContainerCpuLimit), stateMap, pod)
	throttlePod.PodCPUQuota, throttlePod.ContainerCPUQuotas = executor.GetPodUsage(string(stypes.MetricNameContainerCpuQuota), stateMap, pod)
	throttlePod.PodCPUPeriod, throttlePod.ContainerCPUPeriods = executor.GetPodUsage(string(stypes.MetricNameContainerCpuPeriod), stateMap, pod)
	throttlePod.PodQOSPriority = qosPriority

	return throttlePod
}

func combineThrottleDuplicate(e *executor.ThrottleExecutor, throttlePods, throttleUpPods executor.ThrottlePods) {
	for _, t := range throttlePods {
		if i := e.ThrottleDownPods.Find(t.PodTypes); i == -1 {
			e.ThrottleDownPods = append(e.ThrottleDownPods, t)
		} else {
			if t.CPUThrottle.MinCPURatio > e.ThrottleDownPods[i].CPUThrottle.MinCPURatio {
				e.ThrottleDownPods[i].CPUThrottle.MinCPURatio = t.CPUThrottle.MinCPURatio
			}

			if t.CPUThrottle.StepCPURatio > e.ThrottleDownPods[i].CPUThrottle.StepCPURatio {
				e.ThrottleDownPods[i].CPUThrottle.StepCPURatio = t.CPUThrottle.StepCPURatio
			}
		}
	}
	for _, t := range throttleUpPods {

		if i := e.ThrottleUpPods.Find(t.PodTypes); i == -1 {
			e.ThrottleUpPods = append(e.ThrottleUpPods, t)
		} else {
			if t.CPUThrottle.MinCPURatio > e.ThrottleUpPods[i].CPUThrottle.MinCPURatio {
				e.ThrottleUpPods[i].CPUThrottle.MinCPURatio = t.CPUThrottle.MinCPURatio
			}

			if t.CPUThrottle.StepCPURatio > e.ThrottleUpPods[i].CPUThrottle.StepCPURatio {
				e.ThrottleUpPods[i].CPUThrottle.StepCPURatio = t.CPUThrottle.StepCPURatio
			}
		}
	}
}

func combineEvictDuplicate(e *executor.EvictExecutor, evictPods executor.EvictPods) {
	for _, ep := range evictPods {
		if i := e.EvictPods.Find(ep.PodKey); i == -1 {
			e.EvictPods = append(e.EvictPods, ep)
		} else {
			if ep.DeletionGracePeriodSeconds < e.EvictPods[i].DeletionGracePeriodSeconds {
				e.EvictPods[i].DeletionGracePeriodSeconds = ep.DeletionGracePeriodSeconds
			}
		}
	}
}
