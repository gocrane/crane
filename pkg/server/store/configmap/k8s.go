package configmap

import (
	"sync"

	"k8s.io/client-go/kubernetes"

	"github.com/gocrane/crane/pkg/server/store"
)

type K8sStoreConfig struct {
}

type datastore struct {
	client kubernetes.Interface
}

func (ds *datastore) Clusters() store.ClusterStore {
	return NewClusters(ds)
}

var (
	k8sFactory store.Factory
	once       sync.Once
)

func NewK8SStoreFactory(client kubernetes.Interface) (store.Factory, error) {
	once.Do(func() {
		k8sFactory = &datastore{client: client}
	})
	return k8sFactory, nil
}
