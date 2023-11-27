package pod

import (
	"context"
	"fmt"

	"github.com/gocrane/crane/pkg/ensurance/util"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"

	"github.com/gocrane/api/ensurance/v1alpha1"
	"github.com/gocrane/crane/pkg/ensurance/config"
)

type MutatingAdmission struct {
	Config     *config.QOSConfig
	listPodQOS func() ([]*v1alpha1.PodQOS, error)
}

func NewMutatingAdmission(config *config.QOSConfig, listPodQOS func() ([]*v1alpha1.PodQOS, error)) *MutatingAdmission {
	return &MutatingAdmission{
		Config:     config,
		listPodQOS: listPodQOS,
	}
}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (m *MutatingAdmission) Default(ctx context.Context, obj runtime.Object) error {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("expected a Pod but got a %T", obj)
	}

	klog.Infof("mutating started for pod %s/%s", pod.Namespace, pod.Name)

	if m.Config == nil || !m.Config.QOSInitializer.Enable {
		return nil
	}

	if pod.Labels == nil {
		return nil
	}

	ls, err := metav1.LabelSelectorAsSelector(m.Config.QOSInitializer.Selector)
	if err != nil {
		return err
	}

	if !ls.Matches(labels.Set(pod.Labels)) {
		klog.Infof("injection skipped: webhook is not interested in the pod")
		return nil
	}

	qosSlice, err := m.listPodQOS()
	if err != nil {
		return errors.WithMessage(err, "list PodQOS failed")
	}

	/****************************************************************
	 *	Check whether the pod has a low CPUPriority (CPUPriority > 0)
	 ****************************************************************/
	qos := util.MatchPodAndPodQOSSlice(pod, qosSlice)
	if qos == nil {
		klog.Infof("injection skipped: no podqos matched")
		return nil
	}

	if qos.Spec.ResourceQOS.CPUQOS == nil ||
		qos.Spec.ResourceQOS.CPUQOS.CPUPriority == nil ||
		*qos.Spec.ResourceQOS.CPUQOS.CPUPriority == 0 {
		klog.Infof("injection skipped: not a low CPUPriority pod, qos %s", qos.Name)
		return nil
	}
	for _, container := range pod.Spec.InitContainers {
		if container.Name == m.Config.QOSInitializer.InitContainerTemplate.Name {
			klog.Infof("injection skipped: pod has initializerContainer already")
			return nil
		}
	}

	for _, volume := range pod.Spec.Volumes {
		if volume.Name == m.Config.QOSInitializer.VolumeTemplate.Name {
			klog.Infof("injection skipped: pod has initializerVolume already")
			return nil
		}
	}

	if m.Config.QOSInitializer.InitContainerTemplate != nil {
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, *m.Config.QOSInitializer.InitContainerTemplate)
	}

	if m.Config.QOSInitializer.VolumeTemplate != nil {
		pod.Spec.Volumes = append(pod.Spec.Volumes, *m.Config.QOSInitializer.VolumeTemplate)
	}

	klog.Infof("mutating completed for pod %s/%s", pod.Namespace, pod.Name)

	return nil
}
