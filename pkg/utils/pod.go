package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	kubelettypes "k8s.io/kubernetes/pkg/kubelet/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gocrane/crane/pkg/known"
)

const (
	ExtResourcePrefixFormat = "gocrane.io/%s"
)

// GetAvailablePods return a set with pod names that paas IsPodAvailable check
func GetAvailablePods(pods []v1.Pod) []v1.Pod {
	var availablePods []v1.Pod
	timeNow := metav1.Now()

	for _, pod := range pods {
		if IsPodAvailable(&pod, 30, timeNow) {
			availablePods = append(availablePods, pod)
		}
	}
	return availablePods
}

// IsPodAvailable returns true if a pod is available; false otherwise.
// copied from k8s.io/kubernetes/pkg/api/v1/pod.go
func IsPodAvailable(pod *v1.Pod, minReadySeconds int32, now metav1.Time) bool {
	if !IsPodReady(pod) {
		return false
	}

	c := GetPodReadyCondition(pod.Status)
	minReadySecondsDuration := time.Duration(minReadySeconds) * time.Second
	if minReadySeconds == 0 || (!c.LastTransitionTime.IsZero() && c.LastTransitionTime.Add(minReadySecondsDuration).Before(now.Time)) {
		return true
	}
	return false
}

// IsPodReady returns true if a pod is ready; false otherwise.
// copied from k8s.io/kubernetes/pkg/api/v1/pod.go and modified
func IsPodReady(pod *v1.Pod) bool {
	if pod.DeletionTimestamp != nil || pod.Status.Phase != v1.PodRunning {
		return false
	}
	condition := GetPodReadyCondition(pod.Status)
	return condition != nil && condition.Status == v1.ConditionTrue
}

// GetPodReadyCondition extracts the pod ready condition from the given status and returns that.
// Returns nil if the condition is not present.
// copied from k8s.io/kubernetes/pkg/api/v1/pod.go
func GetPodReadyCondition(status v1.PodStatus) *v1.PodCondition {
	_, condition := GetPodCondition(&status, v1.PodReady)
	return condition
}

// GetPodCondition extracts the provided condition from the given status and returns that.
// Returns nil and -1 if the condition is not present, and the index of the located condition.
// copied from k8s.io/kubernetes/pkg/api/v1/pod.go
func GetPodCondition(status *v1.PodStatus, conditionType v1.PodConditionType) (int, *v1.PodCondition) {
	if status == nil {
		return -1, nil
	}
	if status.Conditions == nil {
		return -1, nil
	}
	for i := range status.Conditions {
		if status.Conditions[i].Type == conditionType {
			return i, &status.Conditions[i]
		}
	}
	return -1, nil
}

// EvictPodWithGracePeriod evict pod with grace period
func EvictPodWithGracePeriod(client clientset.Interface, pod *v1.Pod, gracePeriodSeconds *int32) error {
	if kubelettypes.IsCriticalPod(pod) {
		return fmt.Errorf("eviction manager: cannot evict a critical pod(%s)", klog.KObj(pod))
	}

	var grace = GetInt64withDefault(pod.Spec.TerminationGracePeriodSeconds, known.DefaultDeletionGracePeriodSeconds)
	if gracePeriodSeconds != nil {
		grace = int64(*gracePeriodSeconds)
	}

	e := &policyv1beta1.Eviction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
		DeleteOptions: metav1.NewDeleteOptions(grace),
	}

	return client.CoreV1().Pods(pod.Namespace).EvictV1beta1(context.Background(), e)
}

// CalculatePodRequests sum request total from pods
func CalculatePodRequests(pods []v1.Pod, resource v1.ResourceName) (int64, error) {
	var requests int64
	for _, pod := range pods {
		for _, c := range pod.Spec.Containers {
			if containerRequest, ok := c.Resources.Requests[resource]; ok {
				requests += containerRequest.MilliValue()
			} else {
				return 0, fmt.Errorf("missing request for %s", resource)
			}
		}
	}
	return requests, nil
}

// GetPodContainerByName get container info by container name
func GetPodContainerByName(pod *v1.Pod, containerName string) (v1.Container, error) {
	for _, v := range pod.Spec.Containers {
		if v.Name == containerName {
			return v, nil
		}
	}

	return v1.Container{}, fmt.Errorf("container not found")
}

// CalculatePodTemplateRequests sum request total from podTemplate
func CalculatePodTemplateRequests(podTemplate *v1.PodTemplateSpec, resource v1.ResourceName) (int64, error) {
	var requests int64
	for _, c := range podTemplate.Spec.Containers {
		if containerRequest, ok := c.Resources.Requests[resource]; ok {
			requests += containerRequest.MilliValue()
		} else {
			return 0, fmt.Errorf("missing request for %s", resource)
		}
	}

	return requests, nil
}

// GetExtCpuRes get container's gocrane.io/cpu usage
func GetExtCpuRes(container v1.Container) (resource.Quantity, bool) {
	for res, val := range container.Resources.Limits {
		if strings.HasPrefix(res.String(), fmt.Sprintf(ExtResourcePrefixFormat, v1.ResourceCPU)) && val.Value() != 0 {
			return val, true
		}
	}
	return resource.Quantity{}, false
}

func GetContainerNameFromPod(pod *v1.Pod, containerId string) string {
	if containerId == "" {
		return ""
	}

	for _, v := range pod.Status.ContainerStatuses {
		strList := strings.Split(v.ContainerID, "//")
		if len(strList) > 0 {
			if strList[len(strList)-1] == containerId {
				return v.Name
			}
		}
	}

	return ""
}

func GetContainerFromPod(pod *v1.Pod, containerName string) *v1.Container {
	if containerName == "" {
		return nil
	}
	for _, v := range pod.Spec.Containers {
		if v.Name == containerName {
			return &v
		}
	}
	return nil
}

// GetExtCpuRes get container's gocrane.io/cpu usage
func GetContainerExtCpuResFromPod(pod *v1.Pod, containerName string) (resource.Quantity, bool) {
	c := GetContainerFromPod(pod, containerName)
	if c == nil {
		return resource.Quantity{}, false
	}
	return GetExtCpuRes(*c)
}

func GetContainerStatus(pod *v1.Pod, container v1.Container) v1.ContainerState {
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Name == container.Name {
			return cs.State
		}
	}
	return v1.ContainerState{}
}

func GetContainerIdFromPod(pod *v1.Pod, containerName string) string {
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Name == containerName {
			return GetContainerIdFromKey(cs.ContainerID)
		}
	}
	return ""
}

// GetElasticResourceLimit sum all containers resources limit for gocrane.io/resource
// As extended resource is not over committable resource, so request = limit
func GetElasticResourceLimit(pod *v1.Pod, resName v1.ResourceName) (amount int64) {
	resPrefix := fmt.Sprintf(ExtResourcePrefixFormat, resName)
	for i := range pod.Spec.Containers {
		container := pod.Spec.Containers[i]
		for res, val := range container.Resources.Limits {
			if strings.HasPrefix(res.String(), resPrefix) {
				amount += val.MilliValue()
			}
		}
	}
	return
}

func GetDaemonSetPods(kubeClient client.Client, namespace string, name string) ([]corev1.Pod, error) {
	ds := appsv1.DaemonSet{}
	err := kubeClient.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: name}, &ds)
	if err != nil {
		return nil, err
	}

	opts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(ds.Spec.Selector.MatchLabels),
	}

	podList := &corev1.PodList{}
	err = kubeClient.List(context.TODO(), podList, opts...)
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}
