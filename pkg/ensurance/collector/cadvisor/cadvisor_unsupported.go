//go:build !linux
// +build !linux

package cadvisor

import (
	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/collector/types"
	corelisters "k8s.io/client-go/listers/core/v1"
)

type CadvisorCollector struct {
}

func NewCadvisor(_ corelisters.PodLister) *CadvisorCollector {
	return &CadvisorCollector{}
}

func (c *CadvisorCollector) Stop() error {
	return nil
}

func (c *CadvisorCollector) GetType() types.CollectType {
	return types.CadvisorCollectorType
}

func (c *CadvisorCollector) Collect() (map[string][]common.TimeSeries, error) {
	return nil, nil
}
