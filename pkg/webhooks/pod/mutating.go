package pod

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"

	"github.com/gocrane/api/ensurance/v1alpha1"

	"github.com/gocrane/crane/pkg/ensurance/config"
	"github.com/gocrane/crane/pkg/ensurance/util"
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

	klog.V(2).Infof("Mutating started for pod %s/%s", pod.Namespace, pod.Name)

	if !m.available() {
		return nil
	}

	ls, err := metav1.LabelSelectorAsSelector(m.Config.QOSInitializer.Selector)
	if err != nil {
		return err
	}

	if !ls.Matches(labels.Set(pod.Labels)) {
		klog.V(2).Infof("Injection skipped: webhook is not interested in the pod")
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
		klog.V(2).Infof("Injection skipped: no podqos matched")
		return nil
	}

	if qos.Spec.ResourceQOS.CPUQOS == nil ||
		qos.Spec.ResourceQOS.CPUQOS.CPUPriority == nil ||
		*qos.Spec.ResourceQOS.CPUQOS.CPUPriority == 0 {
		klog.V(2).Infof("Injection skipped: not a low CPUPriority pod, qos %s", qos.Name)
		return nil
	}

	for _, container := range pod.Spec.InitContainers {
		if container.Name == m.Config.QOSInitializer.InitContainerTemplate.Name {
			klog.V(2).Infof("Injection skipped: pod has initializerContainer already")
			return nil
		}
	}

	for _, volume := range pod.Spec.Volumes {
		if volume.Name == m.Config.QOSInitializer.VolumeTemplate.Name {
			klog.V(2).Infof("Injection skipped: pod has initializerVolume already")
			return nil
		}
	}

	if m.Config.QOSInitializer.InitContainerTemplate != nil {
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, *m.Config.QOSInitializer.InitContainerTemplate)
	}

	if m.Config.QOSInitializer.VolumeTemplate != nil {
		pod.Spec.Volumes = append(pod.Spec.Volumes, *m.Config.QOSInitializer.VolumeTemplate)
	}

	klog.V(2).Infof("Mutating completed for pod %s/%s", pod.Namespace, pod.Name)

	return nil
}

func (m *MutatingAdmission) available() bool {
	return m.Config != nil &&
		m.Config.QOSInitializer.Enable &&
		m.Config.QOSInitializer.InitContainerTemplate != nil &&
		m.Config.QOSInitializer.VolumeTemplate != nil
}
