package nodelocal

import (
	"github.com/gocrane/crane/pkg/common"
)

type nodeLocalCollector interface {
	name() string
	collect() (map[string][]common.TimeSeries, error)
}
