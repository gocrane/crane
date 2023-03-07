package pod

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/ensurance/config"
)

type MutatingAdmission struct {
	Config *config.QOSConfig
}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (m *MutatingAdmission) Default(ctx context.Context, obj runtime.Object) error {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("expected a Pod but got a %T", obj)
	}
	klog.Infof("Into Pod injection %s/%s", pod.Namespace, pod.Name)

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

	if ls.Matches(labels.Set(pod.Labels)) {
		if m.Config.QOSInitializer.InitContainerTemplate != nil {
			pod.Spec.InitContainers = append(pod.Spec.InitContainers, *m.Config.QOSInitializer.InitContainerTemplate)
		}

		if m.Config.QOSInitializer.VolumeTemplate != nil {
			pod.Spec.Volumes = append(pod.Spec.Volumes, *m.Config.QOSInitializer.VolumeTemplate)
		}

		klog.Infof("Injected QOSInitializer for Pod %s/%s", pod.Namespace, pod.Name)
	}

	return nil
}
