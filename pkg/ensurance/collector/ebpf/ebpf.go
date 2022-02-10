package ebpf

import (
	"sync"

	"github.com/gocrane/crane/pkg/ensurance/collector/types"

	"github.com/gocrane/crane/pkg/common"
)

type EBPF struct {
	name        types.CollectType
	StatusCache sync.Map
}

func NewEBPF() *EBPF {
	e := EBPF{
		name:        types.EbpfCollectorType,
		StatusCache: sync.Map{},
	}
	return &e
}

func (e *EBPF) GetType() types.CollectType {
	return e.name
}

func (e *EBPF) Collect() (map[string][]common.TimeSeries, error) {
	return nil, nil
}

func (e *EBPF) Stop() error {
	return nil
}
