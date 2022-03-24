package target

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/scale"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SelectorFetcher gets a labelSelector used to gather Pods controlled by the given targetRef.
type SelectorFetcher interface {
	// Fetch returns a labelSelector used to gather Pods controlled by the given targetRef.
	Fetch(targetRef *corev1.ObjectReference) (labels.Selector, error)
}

const (
	daemonSet             string = "DaemonSet"
	deployment            string = "Deployment"
	replicaSet            string = "ReplicaSet"
	statefulSet           string = "StatefulSet"
	replicationController string = "ReplicationController"
	job                   string = "Job"
	cronJob               string = "CronJob"
)

var wellKnownControllers = sets.NewString(daemonSet, deployment, replicaSet, statefulSet, replicationController, job, cronJob)

// NewSelectorFetcher returns new instance of SelectorFetcher
func NewSelectorFetcher(scheme *runtime.Scheme, restMapper meta.RESTMapper, scaleClient scale.ScalesGetter, kubeClient client.Client) SelectorFetcher {
	return &targetSelectorFetcher{
		Scheme:      scheme,
		RestMapper:  restMapper,
		ScaleClient: scaleClient,
		KubeClient:  kubeClient,
	}
}

type targetSelectorFetcher struct {
	Scheme      *runtime.Scheme
	RestMapper  meta.RESTMapper
	ScaleClient scale.ScalesGetter
	KubeClient  client.Client
}

func (f *targetSelectorFetcher) Fetch(target *corev1.ObjectReference) (labels.Selector, error) {
	if wellKnownControllers.Has(target.Kind) {
		return f.getLabelSelector(target)
	}

	// not on a list of known controllers, use scale sub-resource
	groupVersion, err := schema.ParseGroupVersion(target.APIVersion)
	if err != nil {
		return nil, err
	}
	groupKind := schema.GroupKind{
		Group: groupVersion.Group,
		Kind:  target.Kind,
	}

	selector, err := f.getLabelSelectorFromScale(groupKind, target.Namespace, target.Name)
	if err != nil {
		return nil, fmt.Errorf("Unhandled targetRef %+v,  error: %v", *target, err)
	}
	return selector, nil
}

func (f *targetSelectorFetcher) getLabelSelector(target *corev1.ObjectReference) (labels.Selector, error) {
	unstructured := &unstructured.Unstructured{}
	unstructured.SetKind(target.Kind)
	unstructured.SetAPIVersion(target.APIVersion)

	if err := f.KubeClient.Get(context.TODO(), client.ObjectKey{Namespace: target.Namespace, Name: target.Name}, unstructured); err != nil {
		return nil, err
	}

	switch strings.ToLower(target.Kind) {
	case strings.ToLower(daemonSet):
		var daemonset appsv1.DaemonSet
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), &daemonset); err != nil {
			return nil, err
		}
		return metav1.LabelSelectorAsSelector(daemonset.Spec.Selector)
	case strings.ToLower(deployment):
		var deployment appsv1.Deployment
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), &deployment); err != nil {
			return nil, err
		}
		return metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	case strings.ToLower(statefulSet):
		var sts appsv1.StatefulSet
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), &sts); err != nil {
			return nil, err
		}
		return metav1.LabelSelectorAsSelector(sts.Spec.Selector)
	case strings.ToLower(replicaSet):
		var rs appsv1.ReplicaSet
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), &rs); err != nil {
			return nil, err
		}
		return metav1.LabelSelectorAsSelector(rs.Spec.Selector)
	case strings.ToLower(job):
		var job batchv1.Job
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), &job); err != nil {
			return nil, err
		}
		return metav1.LabelSelectorAsSelector(job.Spec.Selector)
	case strings.ToLower(cronJob):
		var cjob batchv1.CronJob
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), &cjob); err != nil {
			return nil, err
		}
		return metav1.LabelSelectorAsSelector(metav1.SetAsLabelSelector(cjob.Spec.JobTemplate.Spec.Template.Labels))
	case strings.ToLower(replicationController):
		var rc corev1.ReplicationController
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), &rc); err != nil {
			return nil, err
		}
		return metav1.LabelSelectorAsSelector(metav1.SetAsLabelSelector(rc.Spec.Selector))
	}
	return nil, fmt.Errorf("unable to fetch label seletor for %+v", *target)
}

func (f *targetSelectorFetcher) getLabelSelectorFromScale(groupKind schema.GroupKind, namespace, name string) (labels.Selector, error) {
	mappings, err := f.RestMapper.RESTMappings(groupKind)
	if err != nil {
		return nil, err
	}

	var errs []error
	for _, mapping := range mappings {
		groupResource := mapping.Resource.GroupResource()
		scale, err := f.ScaleClient.Scales(namespace).Get(context.TODO(), groupResource, name, metav1.GetOptions{})
		if err == nil {
			if scale.Status.Selector == "" {
				return nil, fmt.Errorf("Resource %s/%s has an empty selector for scale sub-resource", namespace, name)
			}
			selector, err := labels.Parse(scale.Status.Selector)
			if err != nil {
				return nil, err
			}
			return selector, nil
		}
		errs = append(errs, err)
	}
	return nil, fmt.Errorf("%+v", errs)
}
