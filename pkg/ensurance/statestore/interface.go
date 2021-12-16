package statestore

import (
	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/ensurance/manager"
	"github.com/gocrane/crane/pkg/ensurance/statestore/types"
)

type StateStore interface {
	manager.Manager
	List() map[string][]common.TimeSeries
}

type collector interface {
	GetType() types.CollectType
	Collect() (map[string][]common.TimeSeries, error)
}
