package executor

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/ensurance/client"
	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/utils"
)

const (
	DefaultCoolDownSeconds = 300
)

type ScheduleExecutor struct {
	DisableClassAndPriority *ClassAndPriority
	RestoreClassAndPriority *ClassAndPriority
}

type ClassAndPriority struct {
	PodQOSClass        v1.PodQOSClass
	PriorityClassValue int32
}

func (b *ScheduleExecutor) Avoid(ctx *ExecuteContext) error {
	klog.V(6).Info("DisableScheduledExecutor avoid, %v", *b)

	if b.DisableClassAndPriority == nil {
		return nil
	}

	node, err := ctx.NodeLister.Get(ctx.NodeName)
	if err != nil {
		return err
	}

	// update node condition for block scheduled
	if updateNode, needUpdate := client.UpdateNodeConditions(node, v1.NodeCondition{Type: known.EnsuranceAnalyzedPressureConditionKey, Status: v1.ConditionTrue}); needUpdate {
		if err := client.UpdateNodeStatus(ctx.Client, updateNode, nil); err != nil {
			return err
		}
	}

	// update node taint for block scheduled
	if updateNode, needUpdate := client.UpdateNodeTaints(node, v1.Taint{Key: known.EnsuranceAnalyzedPressureTaintKey, Effect: v1.TaintEffectPreferNoSchedule}); needUpdate {
		if err := client.UpdateNode(ctx.Client, updateNode, nil); err != nil {
			return err
		}
	}

	return nil
}

func (b *ScheduleExecutor) Restore(ctx *ExecuteContext) error {
	klog.V(10).Info("DisableScheduledExecutor restore, %v", *b)

	if b.RestoreClassAndPriority == nil {
		return nil
	}

	node, err := ctx.NodeLister.Get(ctx.NodeName)
	if err != nil {
		return err
	}

	// update node condition for restored scheduled
	if updateNode, needUpdate := client.UpdateNodeConditions(node, v1.NodeCondition{Type: known.EnsuranceAnalyzedPressureConditionKey, Status: v1.ConditionFalse}); needUpdate {
		if err := client.UpdateNodeStatus(ctx.Client, updateNode, nil); err != nil {
			return err
		}
	}

	// update node taint for restored scheduled
	if updateNode, needUpdate := client.RemoveNodeTaints(node, v1.Taint{Key: known.EnsuranceAnalyzedPressureTaintKey, Effect: v1.TaintEffectPreferNoSchedule}); needUpdate {
		if err := client.UpdateNode(ctx.Client, updateNode, nil); err != nil {
			return err
		}
	}

	return nil
}

func (s ClassAndPriority) Less(i ClassAndPriority) bool {
	if comparePodQos(s.PodQOSClass, i.PodQOSClass) == 1 {
		return false
	}

	if comparePodQos(s.PodQOSClass, i.PodQOSClass) == -1 {
		return true
	}

	return s.PriorityClassValue < i.PriorityClassValue
}

func (s ClassAndPriority) Greater(i ClassAndPriority) bool {
	if comparePodQos(s.PodQOSClass, i.PodQOSClass) == 1 {
		return true
	}

	if comparePodQos(s.PodQOSClass, i.PodQOSClass) == -1 {
		return false
	}

	return s.PriorityClassValue > i.PriorityClassValue
}

func GetMaxQOSPriority(podLister corelisters.PodLister, podTypes []types.NamespacedName) (types.NamespacedName, ClassAndPriority) {

	var podType types.NamespacedName
	var scheduledQOSPriority ClassAndPriority

	for _, podNamespace := range podTypes {
		if pod, err := podLister.Pods(podNamespace.Namespace).Get(podNamespace.Name); err != nil {
			klog.V(6).Infof("Warning: getMaxQOSPriority get pod %s not found", podNamespace.String())
			continue
		} else {
			var priority = ClassAndPriority{PodQOSClass: pod.Status.QOSClass, PriorityClassValue: utils.GetInt32withDefault(pod.Spec.Priority, 0) - 1}
			if priority.Greater(scheduledQOSPriority) {
				scheduledQOSPriority = priority
				podType = podNamespace
			}
		}
	}

	return podType, scheduledQOSPriority
}

// We defined guaranteed is the highest qos class, burstable is the middle level
// bestEffort is the lowest
// if a qos class is greater than b, return 1
// if a qos class is less than b, return -1
// if a qos class equal with b , return 0
func comparePodQos(a v1.PodQOSClass, b v1.PodQOSClass) int32 {
	switch b {
	case v1.PodQOSGuaranteed:
		if a == v1.PodQOSGuaranteed {
			return 0
		} else {
			return -1
		}
	case v1.PodQOSBurstable:
		if a == v1.PodQOSGuaranteed {
			return 1
		} else if a == v1.PodQOSBurstable {
			return 0
		} else {
			return -1
		}
	case v1.PodQOSBestEffort:
		if (a == v1.PodQOSGuaranteed) || (a == v1.PodQOSBurstable) {
			return 1
		} else if a == v1.PodQOSBestEffort {
			return 0
		} else {
			return -1
		}
	default:
		if (a == v1.PodQOSGuaranteed) || (a == v1.PodQOSBurstable) || (a == v1.PodQOSBestEffort) {
			return 1
		} else {
			return 0
		}
	}
}
