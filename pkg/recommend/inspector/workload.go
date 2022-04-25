package inspector

import (
	"fmt"
	"strconv"

	"github.com/gocrane/crane/pkg/recommend/types"
)

type WorkloadInspector struct {
	Context *types.Context
}

func (i *WorkloadInspector) Inspect() error {
	deploymentMinReplicas, err := strconv.ParseInt(i.Context.ConfigProperties["ehpa.deployment-min-replicas"], 10, 32)
	if err != nil {
		return err
	}

	statefulsetMinReplicas, err := strconv.ParseInt(i.Context.ConfigProperties["ehpa.statefulset-min-replicas"], 10, 32)
	if err != nil {
		return err
	}

	workloadMinReplicas, err := strconv.ParseInt(i.Context.ConfigProperties["ehpa.workload-min-replicas"], 10, 32)
	if err != nil {
		return err
	}

	if i.Context.Deployment != nil && *i.Context.Deployment.Spec.Replicas < int32(deploymentMinReplicas) {
		return fmt.Errorf("deployment replicas %d should be larger than %d ", *i.Context.Deployment.Spec.Replicas, int32(deploymentMinReplicas))
	}

	if i.Context.StatefulSet != nil && *i.Context.StatefulSet.Spec.Replicas < int32(statefulsetMinReplicas) {
		return fmt.Errorf("statefulSet replicas %d should be larger than %d ", *i.Context.StatefulSet.Spec.Replicas, int32(statefulsetMinReplicas))
	}

	if i.Context.Scale != nil && i.Context.Scale.Spec.Replicas < int32(workloadMinReplicas) {
		return fmt.Errorf("workload replicas %d should be larger than %d ", i.Context.Scale.Spec.Replicas, int32(workloadMinReplicas))
	}

	for _, container := range i.Context.PodTemplate.Spec.Containers {
		if container.Resources.Requests.Cpu() == nil {
			return fmt.Errorf("container %s resource cpu request is empty ", container.Name)
		}

		if container.Resources.Limits.Cpu() == nil {
			return fmt.Errorf("container %s resource cpu limit is empty ", container.Name)
		}
	}

	return nil
}

func (i *WorkloadInspector) Name() string {
	return "WorkloadInspector"
}
