package store

import (
	"fmt"
	"sync"

	_ "github.com/golang/mock/gomock"
)

//go:generate mockgen --build_flags=--mod=mod -self_package=github.com/gocrane/crane/pkg/server/store -destination mock_store.go -package store github.com/gocrane/crane/pkg/server/store Store,ClusterStore

type Store interface {
	Clusters() ClusterStore
}

type Factory func(cfg interface{}) (Store, error)

var storeFactories = make(map[string]Factory)
var factoryLock sync.Mutex

func RegisterFactory(storeType string, factoryFunc Factory) {
	factoryLock.Lock()
	defer factoryLock.Unlock()

	storeFactories[storeType] = factoryFunc
}

func InitStore(storeType string, cfg interface{}) (Store, error) {
	factoryLock.Lock()
	defer factoryLock.Unlock()

	f, ok := storeFactories[storeType]
	if !ok {
		return nil, fmt.Errorf("not supported store type %v", storeType)
	}
	return f(cfg)
}

func GetStore(storeType string) (Store, error) {
	factoryLock.Lock()
	defer factoryLock.Unlock()

	if factoryFunc, ok := storeFactories[storeType]; ok {
		return factoryFunc(storeType)
	} else {
		return nil, fmt.Errorf("not supported store type %v", storeType)
	}
}
