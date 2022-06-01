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
	workloadMinReplicas, err := strconv.ParseInt(i.Context.ConfigProperties["replicas.workload-min-replicas"], 10, 32)
	if err != nil {
		return err
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
