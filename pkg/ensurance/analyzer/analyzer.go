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
	ecache "github.com/gocrane/crane/pkg/ensurance/cache"
	"github.com/gocrane/crane/pkg/ensurance/collector"
	stypes "github.com/gocrane/crane/pkg/ensurance/collector/types"
	"github.com/gocrane/crane/pkg/ensurance/executor"
	"github.com/gocrane/crane/pkg/ensurance/logic"
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

	stateStore *collector.StateStore
	recorder   record.EventRecorder
	actionCh   chan<- executor.AvoidanceExecutor

	logic             logic.Logic
	status            map[string][]common.TimeSeries
	reached           map[string]uint64
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
	stateStore *collector.StateStore,
	noticeCh chan<- executor.AvoidanceExecutor,
) *AnormalyAnalyzer {

	basicLogic := logic.NewBasicLogic()
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "crane-agent"})
	return &AnormalyAnalyzer{
		nodeName:              nodeName,
		logic:                 basicLogic,
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
		stateStore:            stateStore,
		reached:               make(map[string]uint64),
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
		updateTicker := time.NewTicker(10 * time.Second)
		defer updateTicker.Stop()
		for {
			select {
			case <-updateTicker.C:
				s.Analyze()
			case <-stop:
				klog.Infof("AnormalyAnalyzer exit")
				return
			}
		}
	}()

	return
}

func (s *AnormalyAnalyzer) Analyze() {
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
		if matched, err := utils.LabelSelectorMatched(node.Labels, &nep.Spec.Selector); err != nil || !matched {
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

	s.status = s.stateStore.List()
	klog.V(6).Infof("Analyze state store status %#v", s.status)

	// step 2: do analyze for neps
	var dcs []ecache.DetectionCondition
	for _, n := range neps {
		for _, v := range n.Spec.ObjectiveEnsurances {
			var key = strings.Join([]string{n.Name, v.Name}, ".")
			detection, err := s.analyze(key, v)
			if err != nil {
				klog.Errorf("Failed to analyze, %v.", err)
			}
			detection.Nep = n
			dcs = append(dcs, detection)
		}
	}

	klog.V(10).Infof("Analyze dcs: %v", dcs)

	//step 3 : merge
	avoidanceAction := s.merge(avoidanceMaps, dcs)
	if err != nil {
		klog.Errorf("Failed to merge, %v.", err)
		return
	}

	//step 4 :notice the enforcer manager
	s.notify(avoidanceAction)

	return
}

func (s *AnormalyAnalyzer) analyze(key string, object ensuranceapi.ObjectiveEnsurance) (ecache.DetectionCondition, error) {
	var dc = ecache.DetectionCondition{Strategy: object.Strategy, ObjectiveEnsuranceName: object.Name, ActionName: object.AvoidanceActionName}

	if strings.HasPrefix(object.MetricRule.Name, "container") {
		//step1: get series from value
		series := s.getTimeSeriesFromMap(object.MetricRule.Name, object.MetricRule.Selector)

		if len(series) == 0 {
			return dc, fmt.Errorf("metric %s not found", object.MetricRule.Name)
		}

		var basicThrottleQosPriority = executor.ClassAndPriority{PodQOSClass: v1.PodQOSBestEffort, PriorityClassValue: 0}

		var impacted []types.NamespacedName
		var threshold bool

		for _, ts := range series {
			b, err := s.logic.EvalWithMetric(object.MetricRule.Name, float64(object.MetricRule.Value.Value()), ts.Samples[0].Value)
			if err != nil {
				return dc, err
			}

			klog.V(6).Infof("analyze: b %v, Value: %.2f, %s/%s", b, ts.Samples[0].Value,
				common.GetValueByName(ts.Labels, common.LabelNamePodNamespace),
				common.GetValueByName(ts.Labels, common.LabelNamePodName))
			if !b {
				continue
			}

			if (common.GetValueByName(ts.Labels, common.LabelNamePodName) != "") &&
				(common.GetValueByName(ts.Labels, common.LabelNamePodNamespace) != "") {
				pod, err := s.podLister.Pods(common.GetValueByName(ts.Labels, common.LabelNamePodNamespace)).Get(common.GetValueByName(ts.Labels, common.LabelNamePodName))
				if err != nil {
					klog.V(4).Infof("Warning: analyze: Pod %s/%s not found", common.GetValueByName(ts.Labels, common.LabelNamePodNamespace), common.GetValueByName(ts.Labels, common.LabelNamePodName))
					continue
				} else {
					var qosPriority = executor.ClassAndPriority{PodQOSClass: pod.Status.QOSClass, PriorityClassValue: utils.GetInt32withDefault(pod.Spec.Priority, 0)}
					if qosPriority.Greater(basicThrottleQosPriority) {
						impacted = append(impacted, types.NamespacedName{Name: common.GetValueByName(ts.Labels, common.LabelNamePodName),
							Namespace: common.GetValueByName(ts.Labels, common.LabelNamePodNamespace)})
						threshold = true
					}
				}
			} else {
				// node metrics
				klog.Infof("PodName or PodNamespace is empty")
				threshold = true
			}
		}

		klog.V(4).Infof("DoAnalyze: key %s, threshold %v", key, threshold)

		//step3: check is reached action or restored, set the detection
		if threshold {
			s.restored[key] = 0
			reached := utils.GetUint64FromMaps(key, s.reached)
			reached++
			s.reached[key] = reached
			if reached >= uint64(object.AvoidanceThreshold) {
				dc.Triggered = true
				dc.BeInfluencedPods = impacted
			}
		} else {
			s.reached[key] = 0
			restored := utils.GetUint64FromMaps(key, s.restored)
			restored++
			s.restored[key] = restored
			if restored >= uint64(object.RestoreThreshold) {
				dc.Restored = true
			}
		}

	} else {

		//step1: get metric value
		value, err := s.getMetricFromMap(object.MetricRule.Name, object.MetricRule.Selector)
		if err != nil {
			return dc, err
		}

		//step2: use opa to check if reached
		threshold, err := s.logic.EvalWithMetric(object.MetricRule.Name, float64(object.MetricRule.Value.Value()), value)
		if err != nil {
			return dc, err
		}

		//step3: check is reached action or restored, set the detection
		if threshold {
			s.restored[key] = 0
			reached := utils.GetUint64FromMaps(key, s.reached)
			reached++
			s.reached[key] = reached
			if reached >= uint64(object.AvoidanceThreshold) {
				dc.Triggered = true
			}
		} else {
			s.reached[key] = 0
			restored := utils.GetUint64FromMaps(key, s.restored)
			restored++
			s.restored[key] = restored
			if restored >= uint64(object.RestoreThreshold) {
				dc.Restored = true
			}
		}
	}

	return dc, nil
}

func (s *AnormalyAnalyzer) merge(avoidanceMaps map[string]*ensuranceapi.AvoidanceAction, dcs []ecache.DetectionCondition) executor.AvoidanceExecutor {
	var now = time.Now()

	//step1 filter the only dryRun detection
	var dcsFiltered []ecache.DetectionCondition
	for _, dc := range dcs {
		s.logEvent(dc, now)
		if !(dc.Strategy == ensuranceapi.AvoidanceActionStrategyPreview) {
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
		ae.ScheduleExecutor.DisableClassAndPriority = &executor.ClassAndPriority{PodQOSClass: v1.PodQOSBestEffort, PriorityClassValue: 0}
		s.lastTriggeredTime = now
	} else {
		if len(dcsFiltered) == 0 {
			restoreScheduled = true
		} else {
			for _, dc := range dcsFiltered {
				action, ok := avoidanceMaps[dc.ActionName]
				if !ok {
					klog.Warningf("DoMerge for detection,but the action %s not found", dc.ActionName)
					continue
				}

				if dc.Restored {
					var schedulingCoolDown = utils.GetInt64withDefault(action.Spec.CoolDownSeconds, executor.DefaultCoolDownSeconds)
					if now.After(s.lastTriggeredTime.Add(time.Duration(schedulingCoolDown) * time.Second)) {
						restoreScheduled = true
						break
					}
				}
			}
		}
	}

	if restoreScheduled {
		ae.ScheduleExecutor.RestoreClassAndPriority = &executor.ClassAndPriority{PodQOSClass: v1.PodQOSBestEffort, PriorityClassValue: 0}
	}

	//step3 do Throttle merge FilterAndSortThrottlePods
	var throttlePods executor.ThrottlePods
	for _, dc := range dcsFiltered {
		if dc.Triggered {
			action, ok := avoidanceMaps[dc.ActionName]
			if !ok {
				klog.Warningf("The action %s not found.", dc.ActionName)
				continue
			}

			if action.Spec.Throttle != nil {
				var basicThrottleQosPriority executor.ClassAndPriority
				if len(dc.BeInfluencedPods) > 0 {
					_, beInfluencedPodPriority := executor.GetMaxQOSPriority(s.podLister, dc.BeInfluencedPods)
					if beInfluencedPodPriority.Greater(basicThrottleQosPriority) {
						basicThrottleQosPriority = beInfluencedPodPriority
					}
				} else {
					basicThrottleQosPriority = executor.ClassAndPriority{PodQOSClass: v1.PodQOSBestEffort, PriorityClassValue: 0}
				}

				allPods, err := s.podLister.List(labels.Everything())
				if err != nil {
					klog.Errorf("Failed to list all pods: %v", err)
					continue
				}

				for _, v := range allPods {

					var qosPriority = executor.ClassAndPriority{PodQOSClass: v.Status.QOSClass, PriorityClassValue: utils.GetInt32withDefault(v.Spec.Priority, 0)}
					if !qosPriority.Greater(basicThrottleQosPriority) {
						var throttlePod executor.ThrottlePod
						throttlePod.PodTypes = types.NamespacedName{Namespace: v.Namespace, Name: v.Name}
						throttlePod.CPUThrottle.MinCPURatio = action.Spec.Throttle.CPUThrottle.MinCPURatio
						throttlePod.CPUThrottle.StepCPURatio = action.Spec.Throttle.CPUThrottle.StepCPURatio

						throttlePod.PodCPUUsage, throttlePod.ContainerCPUUsages = s.getPodUsage(string(stypes.MetricNameContainerCpuTotalUsage), v)
						throttlePod.PodCPUShare, throttlePod.ContainerCPUShares = s.getPodUsage(string(stypes.MetricNameContainerCpuLimit), v)
						throttlePod.PodCPUQuota, throttlePod.ContainerCPUQuotas = s.getPodUsage(string(stypes.MetricNameContainerCpuQuota), v)
						throttlePod.PodCPUPeriod, throttlePod.ContainerCPUPeriods = s.getPodUsage(string(stypes.MetricNameContainerCpuPeriod), v)
						throttlePod.PodQOSPriority = qosPriority
						throttlePods = append(throttlePods, throttlePod)
					}
				}
			}
		}
	}

	// combine the replicated pod
	for _, t := range throttlePods {
		if i := ae.ThrottleExecutor.ThrottleDownPods.Find(t.PodTypes); i == -1 {
			ae.ThrottleExecutor.ThrottleDownPods = append(ae.ThrottleExecutor.ThrottleDownPods, t)
		} else {
			if t.CPUThrottle.MinCPURatio > ae.ThrottleExecutor.ThrottleDownPods[i].CPUThrottle.MinCPURatio {
				ae.ThrottleExecutor.ThrottleDownPods[i].CPUThrottle.MinCPURatio = t.CPUThrottle.MinCPURatio
			}

			if t.CPUThrottle.StepCPURatio > ae.ThrottleExecutor.ThrottleDownPods[i].CPUThrottle.StepCPURatio {
				ae.ThrottleExecutor.ThrottleDownPods[i].CPUThrottle.StepCPURatio = t.CPUThrottle.StepCPURatio
			}
		}
	}

	// sort the throttle executor by pod qos priority
	sort.Sort(ae.ThrottleExecutor.ThrottleDownPods)

	if restoreScheduled {
		var throttleUpPods executor.ThrottlePods

		for _, dc := range dcsFiltered {
			if dc.Restored {
				action, ok := avoidanceMaps[dc.ActionName]
				if !ok {
					klog.Warningf("DoMerge for detection,but the action %s not found", dc.ActionName)
					continue
				}

				if action.Spec.Throttle != nil {
					allPods, err := s.podLister.List(labels.Everything())
					if err != nil {
						klog.Errorf("Failed to list all pods: %v", err)
						continue
					}

					for _, v := range allPods {
						var qosPriority = executor.ClassAndPriority{PodQOSClass: v.Status.QOSClass, PriorityClassValue: utils.GetInt32withDefault(v.Spec.Priority, 0)}
						var throttlePod executor.ThrottlePod
						throttlePod.PodTypes = types.NamespacedName{Namespace: v.Namespace, Name: v.Name}
						throttlePod.CPUThrottle.MinCPURatio = action.Spec.Throttle.CPUThrottle.MinCPURatio
						throttlePod.CPUThrottle.StepCPURatio = action.Spec.Throttle.CPUThrottle.StepCPURatio
						throttlePod.PodCPUUsage, throttlePod.ContainerCPUUsages = s.getPodUsage(string(stypes.MetricNameContainerCpuTotalUsage), v)
						throttlePod.PodCPUShare, throttlePod.ContainerCPUShares = s.getPodUsage(string(stypes.MetricNameContainerCpuLimit), v)
						throttlePod.PodCPUQuota, throttlePod.ContainerCPUQuotas = s.getPodUsage(string(stypes.MetricNameContainerCpuQuota), v)
						throttlePod.PodCPUPeriod, throttlePod.ContainerCPUPeriods = s.getPodUsage(string(stypes.MetricNameContainerCpuPeriod), v)
						throttlePod.PodQOSPriority = qosPriority
						throttleUpPods = append(throttleUpPods, throttlePod)
					}
				}
			}
		}

		// combine the replicated pod
		for _, t := range throttleUpPods {

			if i := ae.ThrottleExecutor.ThrottleUpPods.Find(t.PodTypes); i == -1 {
				ae.ThrottleExecutor.ThrottleUpPods = append(ae.ThrottleExecutor.ThrottleUpPods, t)
			} else {
				if t.CPUThrottle.MinCPURatio > ae.ThrottleExecutor.ThrottleUpPods[i].CPUThrottle.MinCPURatio {
					ae.ThrottleExecutor.ThrottleUpPods[i].CPUThrottle.MinCPURatio = t.CPUThrottle.MinCPURatio
				}

				if t.CPUThrottle.StepCPURatio > ae.ThrottleExecutor.ThrottleUpPods[i].CPUThrottle.StepCPURatio {
					ae.ThrottleExecutor.ThrottleUpPods[i].CPUThrottle.StepCPURatio = t.CPUThrottle.StepCPURatio
				}
			}
		}

		// sort the throttle executor by pod qos priority()
		sort.Sort(sort.Reverse(ae.ThrottleExecutor.ThrottleUpPods))
	}

	//step4 do Evict merge  FilterAndSortEvictPods
	var basicEvictQosPriority = executor.ClassAndPriority{PodQOSClass: v1.PodQOSBestEffort, PriorityClassValue: 0}
	var evictPods executor.EvictPods
	for _, dc := range dcsFiltered {
		if dc.Triggered {
			action, ok := avoidanceMaps[dc.ActionName]
			if !ok {
				klog.Warningf("DoMerge for detection,but the action %s not found", dc.ActionName)
				continue
			}

			if action.Spec.Eviction != nil {
				var deletionGracePeriodSeconds = utils.GetInt32withDefault(action.Spec.Eviction.TerminationGracePeriodSeconds, executor.DefaultDeletionGracePeriodSeconds)
				allPods, err := s.podLister.List(labels.Everything())
				if err != nil {
					klog.Errorf("Failed to list all pods: %v.", err)
					continue
				}

				for _, v := range allPods {
					var classAndPriority = executor.ClassAndPriority{PodQOSClass: v.Status.QOSClass, PriorityClassValue: utils.GetInt32withDefault(v.Spec.Priority, 0)}
					if !classAndPriority.Greater(basicEvictQosPriority) {
						evictPods = append(evictPods, executor.EvictPod{DeletionGracePeriodSeconds: deletionGracePeriodSeconds,
							PodKey: types.NamespacedName{Name: v.Name, Namespace: v.Namespace}, ClassAndPriority: classAndPriority})
					}
				}
			}
		}
	}

	// remove the replicated pod
	for _, e := range evictPods {
		if i := ae.EvictExecutor.EvictPods.Find(e.PodKey); i == -1 {
			ae.EvictExecutor.EvictPods = append(ae.EvictExecutor.EvictPods, e)
		} else {
			if e.DeletionGracePeriodSeconds < ae.EvictExecutor.EvictPods[i].DeletionGracePeriodSeconds {
				ae.EvictExecutor.EvictPods[i].DeletionGracePeriodSeconds = e.DeletionGracePeriodSeconds
			} else {
				// nothing to do,replicated filter the evictPod
			}
		}
	}

	// sort the evicting executor by pod qos priority
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

func (s *AnormalyAnalyzer) getMetricFromMap(metricName string, selector *metav1.LabelSelector) (float64, error) {
	// step1: get the value from map
	if v, ok := s.status[metricName]; ok {
		for _, vv := range v {
			if matched, err := utils.LabelSelectorMatched(common.Labels2Maps(vv.Labels), selector); err != nil {
				return 0, err
			} else if !matched {
				continue
			} else {
				return vv.Samples[0].Value, nil
			}
		}
	}

	return 0, fmt.Errorf("metricName %s not found value", metricName)
}

func (s *AnormalyAnalyzer) getTimeSeriesFromMap(metricName string, selector *metav1.LabelSelector) []common.TimeSeries {
	var series []common.TimeSeries

	// step1: get the series from maps
	if v, ok := s.status[metricName]; ok {
		for _, vv := range v {
			if matched, err := utils.LabelSelectorMatched(common.Labels2Maps(vv.Labels), selector); err != nil {
				continue
			} else if !matched {
				continue
			} else {
				series = append(series, vv)
			}
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

func (s *AnormalyAnalyzer) getPodUsage(metricName string, pod *v1.Pod) (float64, []executor.ContainerUsage) {
	var podUsage = 0.0
	var containerUsages []executor.ContainerUsage

	var podMaps = map[string]string{common.LabelNamePodName: pod.Name, common.LabelNamePodNamespace: pod.Namespace, common.LabelNamePodUid: string(pod.UID)}
	if v, ok := s.status[metricName]; ok {
		for _, vv := range v {
			var labelMaps = common.Labels2Maps(vv.Labels)
			if utils.ContainMaps(labelMaps, podMaps) {
				if labelMaps[common.LabelNameContainerId] == "" {
					podUsage = vv.Samples[0].Value
				} else {
					containerUsages = append(containerUsages, executor.ContainerUsage{ContainerId: labelMaps[common.LabelNameContainerId],
						ContainerName: labelMaps[common.LabelNameContainerName], Value: vv.Samples[0].Value})
				}
			}
		}
	} else {
		klog.Errorf("getPodUsage %s is not found in the status", metricName)
	}

	return podUsage, containerUsages
}
