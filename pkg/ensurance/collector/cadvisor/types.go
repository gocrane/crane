package cadvisor

import (
	info "github.com/google/cadvisor/info/v1"
	cadvisorapiv2 "github.com/google/cadvisor/info/v2"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/collector/types"
)

type Interface interface {
	Stop() error
	GetType() types.CollectType
	Collect() (map[string][]common.TimeSeries, error)
	ContainerInfoV2(containerName string, options cadvisorapiv2.RequestOptions) (map[string]cadvisorapiv2.ContainerInfo, error)
	ContainerInfo(containerName string, query *info.ContainerInfoRequest) (*info.ContainerInfo, error)
}
