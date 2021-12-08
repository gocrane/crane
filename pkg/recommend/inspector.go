package recommend

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	analysisapi "github.com/gocrane/api/analysis/v1alpha1"
	podutil "github.com/gocrane/crane/pkg/utils"
)

type WorkloadInspector struct {
	Context *Context
}

func (i *WorkloadInspector) Inspect() error {
	if i.Context.Deployment != nil && *i.Context.Deployment.Spec.Replicas < recommenderPolicy.Spec.InspectorPolicy.DeploymentMinReplicas {
		return fmt.Errorf("Deployment replicas %d should be larger than %d ", *i.Context.Deployment.Spec.Replicas, recommenderPolicy.Spec.InspectorPolicy.DeploymentMinReplicas)
	}

	if i.Context.StatefulSet != nil && *i.Context.StatefulSet.Spec.Replicas < recommenderPolicy.Spec.InspectorPolicy.StatefulSetMinReplicas {
		return fmt.Errorf("StatefulSet replicas %d should be larger than %d ", *i.Context.StatefulSet.Spec.Replicas, recommenderPolicy.Spec.InspectorPolicy.StatefulSetMinReplicas)
	}

	if i.Context.Scale != nil && i.Context.Scale.Spec.Replicas < recommenderPolicy.Spec.InspectorPolicy.WorkloadMinReplicas {
		return fmt.Errorf("Workload replicas %d should be larger than %d ", i.Context.Scale.Spec.Replicas, recommenderPolicy.Spec.InspectorPolicy.WorkloadMinReplicas)
	}

	return nil
}

type WorkloadPodsInspector struct {
	Context *Context
	Pods    []v1.Pod
}

func (i *WorkloadPodsInspector) Inspect() error {
	if len(i.Pods) == 0 {
		return fmt.Errorf("Existing pods should be larger than 0 ")
	}

	readyPods := 0
	for _, pod := range i.Pods {
		if podutil.IsPodAvailable(&pod, recommenderPolicy.Spec.InspectorPolicy.PodMinReadySeconds, metav1.Now()) {
			readyPods++
		}
	}

	availableRatio := float64(readyPods) / float64(len(i.Pods))
	if availableRatio < recommenderPolicy.Spec.InspectorPolicy.PodAvailableRatio {
		return fmt.Errorf("Pod available ratio is %.3f less than %.3f ", availableRatio, recommenderPolicy.Spec.InspectorPolicy.PodAvailableRatio)
	}

	return nil
}

func NewInspectors(context *Context) []Inspector {
	var inspectors []Inspector

	switch context.Recommendation.Spec.Type {
	case analysisapi.TypeResource:
		// todo
	case analysisapi.TypeHPA:
		if context.Scale != nil {
			inspector := &WorkloadInspector{
				Context: context,
			}
			inspectors = append(inspectors, inspector)
		}

		if context.Pods != nil {
			inspector := &WorkloadPodsInspector{
				Pods:    context.Pods,
				Context: context,
			}
			inspectors = append(inspectors, inspector)
		}
	}

	return inspectors
}
