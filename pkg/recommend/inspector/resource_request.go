package inspector

import (
	"fmt"

	"github.com/gocrane/crane/pkg/recommend/types"
)

type ResourceRequestInspector struct {
	*types.Context
}

func (i *ResourceRequestInspector) Inspect() error {
	if len(i.Pods) == 0 {
		return fmt.Errorf("pod not found")
	}

	pod := i.Pods[0]
	if len(pod.OwnerReferences) == 0 {
		return fmt.Errorf("owner reference not found")
	}

	return nil
}

func (i *ResourceRequestInspector) Name() string {
	return "ResourceRequestInspector"
}
