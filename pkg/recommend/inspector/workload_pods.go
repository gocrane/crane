package inspector

import (
	"fmt"
	"strconv"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gocrane/crane/pkg/recommend/types"
	podutil "github.com/gocrane/crane/pkg/utils"
)

type WorkloadPodsInspector struct {
	Context *types.Context
	Pods    []v1.Pod
}

func (i *WorkloadPodsInspector) Inspect() error {
	if len(i.Pods) == 0 {
		return fmt.Errorf("existing pods should be larger than 0 ")
	}

	podMinReadySeconds, err := strconv.ParseInt(i.Context.ConfigProperties["ehpa.pod-min-ready-seconds"], 10, 32)
	if err != nil {
		return err
	}

	podAvailableRatio, err := strconv.ParseFloat(i.Context.ConfigProperties["ehpa.pod-available-ratio"], 64)
	if err != nil {
		return err
	}

	readyPods := 0
	for _, pod := range i.Pods {
		if podutil.IsPodAvailable(&pod, int32(podMinReadySeconds), metav1.Now()) {
			readyPods++
		}
	}

	if readyPods == 0 {
		return fmt.Errorf("pod available number must larger than zero. ")
	}

	availableRatio := float64(readyPods) / float64(len(i.Pods))
	if availableRatio < podAvailableRatio {
		return fmt.Errorf("pod available ratio is %.3f less than %.3f ", availableRatio, podAvailableRatio)
	}

	i.Context.ReadyPodNumber = readyPods
	return nil
}

func (i *WorkloadPodsInspector) Name() string {
	return "WorkloadPodsInspector"
}
