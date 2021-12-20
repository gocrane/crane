package statestore

import (
	"github.com/gocrane/crane/pkg/common"
)

type StateStore interface {
	List() map[string][]common.TimeSeries
}

/*
type collector interface {
	GetType() types.CollectType
	Collect() (map[string][]common.TimeSeries, error)
}
*/
