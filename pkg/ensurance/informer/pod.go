package informer

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	kubelettypes "k8s.io/kubernetes/pkg/kubelet/types"

	"github.com/gocrane/crane/pkg/utils/log"
)

//EvictPodWithGracePeriod evict pod with grace period
func EvictPodWithGracePeriod(client clientset.Interface, pod *v1.Pod, gracePeriodSeconds int32) error {

	if kubelettypes.IsCriticalPod(pod) {
		return fmt.Errorf("Eviction manager: cannot evict a critical pod(%s)", log.GenerateObj(pod))
	}

	e := &policyv1beta1.Eviction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
		DeleteOptions: metav1.NewDeleteOptions(int64(gracePeriodSeconds)),
	}

	return client.CoreV1().Pods(pod.Namespace).EvictV1beta1(context.Background(), e)
}

func GetAllPodFromInformer(podInformer cache.SharedIndexInformer) []*v1.Pod {
	if podInformer == nil {
		return []*v1.Pod{}
	}

	var podList []*v1.Pod
	allPods := podInformer.GetStore().List()
	for _, p := range allPods {
		pod := p.(*v1.Pod)
		podList = append(podList, pod)
	}

	return podList
}

func GetPodFromInformer(podInformer cache.SharedIndexInformer, key string) (*v1.Pod, error) {
	obj, exited, err := podInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil, err
	}

	if !exited {
		return nil, fmt.Errorf("pod(%s) not found", key)
	}

	// re-assign new pod info
	return obj.(*v1.Pod), nil
}
