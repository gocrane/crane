package collector

import (
	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/collector/types"
)

type Collector interface {
	GetType() types.CollectType
	Collect() (map[string][]common.TimeSeries, error)
}
