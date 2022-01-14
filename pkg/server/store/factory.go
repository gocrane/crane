package store

import _ "github.com/golang/mock/gomock"

//go:generate mockgen --build_flags=--mod=mod -self_package=github.com/gocrane/crane/pkg/server/store -destination mock_store.go -package store github.com/gocrane/crane/pkg/server/store Factory,ClusterStore

type Factory interface {
	Clusters() ClusterStore
}

type CreateOptions struct {
}

type DeleteOptions struct {
}

type UpdateOptions struct {
}

type ListOptions struct {
}

type GetOptions struct {
}

var storeFactory Factory

// GetStoreFactory return the store client factory instance.
func GetStoreFactory() Factory {
	return storeFactory
}

// SetStoreFactory set the crane store factory.
func SetStoreFactory(factory Factory) {
	storeFactory = factory
}
