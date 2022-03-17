//go:build !linux
// +build !linux

package cadvisor

import (
	info "github.com/google/cadvisor/info/v1"
	cadvisorapiv2 "github.com/google/cadvisor/info/v2"
	corelisters "k8s.io/client-go/listers/core/v1"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/collector/types"
)

type CadvisorCollectorUnsupport struct {
}

var _ Interface = new(CadvisorCollectorUnsupport)

func NewCadvisor(_ corelisters.PodLister) Interface {
	return &CadvisorCollectorUnsupport{}
}

func (c *CadvisorCollectorUnsupport) Stop() error {
	return nil
}

func (c *CadvisorCollectorUnsupport) GetType() types.CollectType {
	return types.CadvisorCollectorType
}

func (c *CadvisorCollectorUnsupport) Collect() (map[string][]common.TimeSeries, error) {
	return nil, nil
}

func (c *CadvisorCollectorUnsupport) ContainerInfoV2(containerName string, options cadvisorapiv2.RequestOptions) (map[string]cadvisorapiv2.ContainerInfo, error) {
	return nil, nil
}

func (c *CadvisorCollectorUnsupport) ContainerInfo(string, *info.ContainerInfoRequest) (*info.ContainerInfo, error) {
	return nil, nil
}

func CheckMetricNameExist(name string) bool {
	return false
}
