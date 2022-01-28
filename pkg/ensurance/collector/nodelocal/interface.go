package nodelocal

import (
	corelisters "k8s.io/client-go/listers/core/v1"

	"github.com/gocrane/crane/pkg/common"
)

type nodeLocalCollector interface {
	name() string
	collect() (map[string][]common.TimeSeries, error)
}

type NodeLocalContext struct {
	Ifaces    []string
	PodLister corelisters.PodLister
}
