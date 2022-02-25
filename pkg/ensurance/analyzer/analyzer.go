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
	var dcs []ecache.DetectionCondition
	for _, n := range neps {
		for _, v := range n.Spec.ObjectiveEnsurances {
			var key = strings.Join([]string{n.Name, v.Name}, ".")
			detection, err := s.analyze(key, v, state)
			if err != nil {
				klog.Errorf("Failed to analyze, %v.", err)
			}
			detection.Nep = n
			dcs = append(dcs, detection)
		}
	}

	klog.V(6).Infof("Analyze dcs: %v", dcs)

	//step 3 : merge
	avoidanceAction := s.merge(state, avoidanceMaps, dcs)
	if err != nil {
		klog.Errorf("Failed to merge, %v.", err)
		return
	}

	//step 4 :notice the enforcer manager
	s.notify(avoidanceAction)

	return
}

func (s *AnormalyAnalyzer) getImpacted(ts common.TimeSeries) []types.NamespacedName {
	//TODO: basicThrottleQosPriority be a input para to be a vara in AnormalyAnalyzer
	var basicThrottleQosPriority = executor.ClassAndPriority{PodQOSClass: v1.PodQOSBestEffort, PriorityClassValue: 0}
	var impacted []types.NamespacedName

	pod, err := s.podLister.Pods(common.GetValueByName(ts.Labels, common.LabelNamePodNamespace)).Get(common.GetValueByName(ts.Labels, common.LabelNamePodName))
	if err != nil {
		klog.V(4).Infof("Warning: analyze: Pod %s/%s not found", common.GetValueByName(ts.Labels, common.LabelNamePodNamespace), common.GetValueByName(ts.Labels, common.LabelNamePodName))
		return []types.NamespacedName{}
	}

	var qosPriority = executor.ClassAndPriority{PodQOSClass: pod.Status.QOSClass, PriorityClassValue: utils.GetInt32withDefault(pod.Spec.Priority, 0)}
	if qosPriority.Greater(basicThrottleQosPriority) {
		impacted = append(impacted, types.NamespacedName{Name: common.GetValueByName(ts.Labels, common.LabelNamePodName),
			Namespace: common.GetValueByName(ts.Labels, common.LabelNamePodNamespace)})
	}
	return impacted
}

func (s *AnormalyAnalyzer) isContainerMetric(object ensuranceapi.ObjectiveEnsurance, ts common.TimeSeries) bool {
	return strings.HasPrefix(object.MetricRule.Name, "container") && (common.GetValueByName(ts.Labels, common.LabelNamePodName) != "") &&
		(common.GetValueByName(ts.Labels, common.LabelNamePodNamespace) != "")
}

func (s *AnormalyAnalyzer) getSeries(state []common.TimeSeries, selector *metav1.LabelSelector, metricName string) ([]common.TimeSeries, error) {
	series := s.getTimeSeriesFromMap(state, selector)
	if len(series) == 0 {
		return []common.TimeSeries{}, fmt.Errorf("time series length is 0 for metric %s", metricName)
	}
	return series, nil
}

func (s *AnormalyAnalyzer) trigger(series []common.TimeSeries, object ensuranceapi.ObjectiveEnsurance) (bool, []types.NamespacedName) {
	var impacted []types.NamespacedName
	var triggered, threshold bool
	for _, ts := range series {
		triggered = s.evaluator.EvalWithMetric(object.MetricRule.Name, float64(object.MetricRule.Value.Value()), ts.Samples[0].Value)

		klog.V(6).Infof("Anormaly detection result %v, Name: %s, Value: %.2f, %s/%s", triggered,
			object.MetricRule.Name,
			ts.Samples[0].Value,
			common.GetValueByName(ts.Labels, common.LabelNamePodNamespace),
			common.GetValueByName(ts.Labels, common.LabelNamePodName))

		if triggered {
			if s.isContainerMetric(object, ts) {
				influced := s.getImpacted(ts)
				if len(influced) > 0 {
					threshold = true
					impacted = append(impacted, influced...)
				}
			} else {
				threshold = true
				return threshold, []types.NamespacedName{}
			}
		}
	}
	return threshold, impacted
}

func (s *AnormalyAnalyzer) analyze(key string, object ensuranceapi.ObjectiveEnsurance, stateMap map[string][]common.TimeSeries) (ecache.DetectionCondition, error) {
	var dc = ecache.DetectionCondition{Strategy: object.Strategy, ObjectiveEnsuranceName: object.Name, ActionName: object.AvoidanceActionName}

	state, ok := stateMap[object.MetricRule.Name]
	if !ok {
		return dc, fmt.Errorf("metric %s not found", object.MetricRule.Name)
	}

	//step1: get series from value
	series, err := s.getSeries(state, object.MetricRule.Selector, object.MetricRule.Name)
	if err != nil {
		return dc, err
	}

	//step2: use opa to check if triggered and get impacted pods for container MetricRule
	threshold, impacted := s.trigger(series, object)

	klog.V(4).Infof("DoAnalyze: key %s, threshold %v", key, threshold)

	//step3: check is triggered action or restored, set the detection
	s.actionTriggeredRestored(threshold, key, object, &dc, impacted)

	return dc, nil
}

func (s *AnormalyAnalyzer) actionTriggeredRestored(threshold bool, key string, object ensuranceapi.ObjectiveEnsurance, dc *ecache.DetectionCondition, impacted []types.NamespacedName) {
	if threshold {
		s.restored[key] = 0
		triggered := utils.GetUint64FromMaps(key, s.triggered)
		triggered++
		s.triggered[key] = triggered
		if triggered >= uint64(object.AvoidanceThreshold) {
			dc.Triggered = true
			dc.BeInfluencedPods = impacted
		}
	} else {
		s.triggered[key] = 0
		restored := utils.GetUint64FromMaps(key, s.restored)
		restored++
		s.restored[key] = restored
		if restored >= uint64(object.RestoreThreshold) {
			dc.Restored = true
		}
	}
}

func (s *AnormalyAnalyzer) filterDryRunDetections(dcs []ecache.DetectionCondition) []ecache.DetectionCondition {
	var dcsFiltered []ecache.DetectionCondition
	now := time.Now()
	for _, dc := range dcs {
		s.logEvent(dc, now)
		if !(dc.Strategy == ensuranceapi.AvoidanceActionStrategyPreview) {
			dcsFiltered = append(dcsFiltered, dc)
		}
	}
	return dcsFiltered
}

func (s *AnormalyAnalyzer) merge(stateMap map[string][]common.TimeSeries, avoidanceMaps map[string]*ensuranceapi.AvoidanceAction, dcs []ecache.DetectionCondition) executor.AvoidanceExecutor {
	var ae executor.AvoidanceExecutor

	//step1 filter dry run detections
	var dcsFiltered []ecache.DetectionCondition
	dcsFiltered = s.filterDryRunDetections(dcs)

	//step2 do DisableScheduled merge
	enableSchedule := s.ScheduleMerge(dcsFiltered, avoidanceMaps, &ae)

	for _, dc := range dcsFiltered {
		action, ok := avoidanceMaps[dc.ActionName]
		if !ok {
			klog.Warningf("The action %s not found.", dc.ActionName)
			continue
		}

		//step3 do Throttle merge FilterAndSortThrottlePods
		if action.Spec.Throttle != nil {
			throttlePods, throttleUpPods := s.GetThrottleActionPods(enableSchedule, dc, action, stateMap)
			// combine the replicated pod
			ae.ThrottleExecutor.Deduplicate(throttlePods.(executor.ThrottlePods), throttleUpPods.(executor.ThrottlePods))
		}

		//step4 do Evict merge FilterAndSortEvictPods
		if action.Spec.Eviction != nil {
			evictPods, _ := s.GetEvictActionPods(dc.Triggered, action)
			// remove the replicated pod
			ae.EvictExecutor.Deduplicate(evictPods.(executor.EvictPods), nil)
		}
	}

	// sort the throttle executor by pod qos priority
	sort.Sort(ae.ThrottleExecutor.ThrottleDownPods)
	sort.Sort(sort.Reverse(ae.ThrottleExecutor.ThrottleUpPods))

	// sort the evict executor by pod qos priority
	sort.Sort(ae.EvictExecutor.EvictPods)

	return ae
}

func (s *AnormalyAnalyzer) logEvent(dc ecache.DetectionCondition, now time.Time) {
	var key = strings.Join([]string{dc.Nep.Name, dc.ObjectiveEnsuranceName}, "/")

	if !(dc.Triggered || dc.Restored) {
		return
	}

	nodeRef := utils.GetNodeRef(s.nodeName)

	//step1 print log if the detection state is changed
	//step2 produce event
	if dc.Triggered {
		klog.V(4).Infof("LOG: %s triggered action %s", key, dc.ActionName)

		// record an event about the objective ensurance triggered
		s.recorder.Event(nodeRef, v1.EventTypeWarning, "AvoidanceTriggered", fmt.Sprintf("%s triggered action %s", key, dc.ActionName))
		s.actionEventStatus[key] = ecache.DetectionStatus{IsTriggered: true, LastTime: now}
	}

	if dc.Restored {
		if s.actionTriggered(dc) {
			klog.V(4).Infof("LOG: %s restored action %s", key, dc.ActionName)
			// record an event about the objective ensurance restored
			s.recorder.Event(nodeRef, v1.EventTypeNormal, "RestoreTriggered", fmt.Sprintf("%s restored action %s", key, dc.ActionName))
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

func (s *AnormalyAnalyzer) actionTriggered(dc ecache.DetectionCondition) bool {
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

func (s *AnormalyAnalyzer) GetThrottleActionPods(enableSchedule bool, dc ecache.DetectionCondition,
	action *ensuranceapi.AvoidanceAction, stateMap map[string][]common.TimeSeries) (throttlePodsRet, throttleUpPodsRet interface{}) {

	throttlePods, throttleUpPods := []executor.ThrottlePod{}, []executor.ThrottlePod{}

	allPods, err := s.podLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list all pods: %v", err)
		return
	}

	for _, pod := range allPods {
		if dc.Triggered {
			if smaller, qosPriority := s.SmallerThanThrottleBaseLine(dc, pod); smaller {
				throttlePods = append(throttlePods, throttlePodConstruct(qosPriority, pod, stateMap, action))
			}
		}
		if enableSchedule && dc.Restored {
			for _, pod := range allPods {
				var qosPriority = executor.ClassAndPriority{PodQOSClass: pod.Status.QOSClass, PriorityClassValue: utils.GetInt32withDefault(pod.Spec.Priority, 0)}
				throttleUpPods = append(throttleUpPods, throttlePodConstruct(qosPriority, pod, stateMap, action))
			}
		}
	}

	return throttlePods, throttleUpPods
}

func (s *AnormalyAnalyzer) GetThrottleBaseLine(BeInfluencedPods []types.NamespacedName) executor.ClassAndPriority {
	var basicThrottleQosPriority executor.ClassAndPriority
	if len(BeInfluencedPods) > 0 {
		_, beInfluencedPodPriority := executor.GetMaxQOSPriority(s.podLister, BeInfluencedPods)
		if beInfluencedPodPriority.Greater(basicThrottleQosPriority) {
			basicThrottleQosPriority = beInfluencedPodPriority
		}
	} else {
		//TODO: basicThrottleQosPriority be a input para to be a vara in AnormalyAnalyzer and EvictExecutor
		basicThrottleQosPriority = executor.ClassAndPriority{PodQOSClass: v1.PodQOSBestEffort, PriorityClassValue: 0}
	}
	return basicThrottleQosPriority
}

func (s *AnormalyAnalyzer) SmallerThanThrottleBaseLine(dc ecache.DetectionCondition, pod *v1.Pod) (bool, executor.ClassAndPriority) {
	basicThrottleQosPriority := s.GetThrottleBaseLine(dc.BeInfluencedPods)
	var qosPriority = executor.ClassAndPriority{PodQOSClass: pod.Status.QOSClass, PriorityClassValue: utils.GetInt32withDefault(pod.Spec.Priority, 0)}
	if !qosPriority.Greater(basicThrottleQosPriority) {
		return true, qosPriority
	}
	return false, qosPriority
}

func (s *AnormalyAnalyzer) GetEvictActionPods(triggered bool, action *ensuranceapi.AvoidanceAction) (evictPodsRet, _ interface{}) {
	//TODO: basicEvictQosPriority be a input para to be a vara in AnormalyAnalyzer and EvictExecutor
	var basicEvictQosPriority = executor.ClassAndPriority{PodQOSClass: v1.PodQOSBestEffort, PriorityClassValue: 0}
	evictPods := []executor.EvictPod{}

	if triggered {
		var deletionGracePeriodSeconds = utils.GetInt32withDefault(action.Spec.Eviction.TerminationGracePeriodSeconds, executor.DefaultDeletionGracePeriodSeconds)
		allPods, err := s.podLister.List(labels.Everything())
		if err != nil {
			klog.Errorf("Failed to list all pods: %v.", err)
			return
		}

		for _, v := range allPods {
			var classAndPriority = executor.ClassAndPriority{PodQOSClass: v.Status.QOSClass, PriorityClassValue: utils.GetInt32withDefault(v.Spec.Priority, 0)}
			if !classAndPriority.Greater(basicEvictQosPriority) {
				evictPods = append(evictPods, executor.EvictPod{DeletionGracePeriodSeconds: deletionGracePeriodSeconds,
					PodKey: types.NamespacedName{Name: v.Name, Namespace: v.Namespace}, ClassAndPriority: classAndPriority})
			}
		}
	}
	return evictPods, executor.EvictPods{}
}

func (s *AnormalyAnalyzer) ScheduleMerge(dcsFiltered []ecache.DetectionCondition, avoidanceMaps map[string]*ensuranceapi.AvoidanceAction, ae *executor.AvoidanceExecutor) (enableSchedule bool) {
	var now = time.Now()
	enableSchedule = false
	for _, dc := range dcsFiltered {
		if dc.Triggered {
			enableSchedule = false
		}
		if dc.Restored {
			action, ok := avoidanceMaps[dc.ActionName]
			if !ok {
				klog.Warningf("DoMerge for detection,but the action %s not found", dc.ActionName)
				continue
			}
			var schedulingCoolDown = utils.GetInt64withDefault(action.Spec.CoolDownSeconds, executor.DefaultCoolDownSeconds)
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

func throttlePodConstruct(qosPriority executor.ClassAndPriority, pod *v1.Pod, stateMap map[string][]common.TimeSeries, action *ensuranceapi.AvoidanceAction) executor.ThrottlePod{
	var throttlePod executor.ThrottlePod

	throttlePod.PodTypes = types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}
	throttlePod.CPUThrottle.MinCPURatio = action.Spec.Throttle.CPUThrottle.MinCPURatio
	throttlePod.CPUThrottle.StepCPURatio = action.Spec.Throttle.CPUThrottle.StepCPURatio

	throttlePod.PodCPUUsage, throttlePod.ContainerCPUUsages = executor.GetPodUsage(string(stypes.MetricNameContainerCpuTotalUsage), stateMap, pod)
	throttlePod.PodCPUShare, throttlePod.ContainerCPUShares = executor.GetPodUsage(string(stypes.MetricNameContainerCpuLimit), stateMap, pod)
	throttlePod.PodCPUQuota, throttlePod.ContainerCPUQuotas = executor.GetPodUsage(string(stypes.MetricNameContainerCpuQuota), stateMap, pod)
	throttlePod.PodCPUPeriod, throttlePod.ContainerCPUPeriods = executor.GetPodUsage(string(stypes.MetricNameContainerCpuPeriod), stateMap, pod)
	throttlePod.PodQOSPriority = qosPriority

	return throttlePod
}