package secret

import (
	"context"
	"fmt"
	"sync"

	"github.com/gocrane/crane/pkg/known"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/gocrane/crane/pkg/server/store"
)

type StoreConfig struct {
	Client kubernetes.Interface
}

type datastore struct {
	client kubernetes.Interface
}

func (ds *datastore) Clusters() store.ClusterStore {
	return NewClusters(ds)
}

var (
	k8sFactory store.Store
	once       sync.Once
)

func DefaultSecretStore() *v1.Secret {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CraneStoreClustersSecretName,
			Namespace: known.CraneSystemNamespace,
			Labels:    map[string]string{CraneStoreLabel: "true"},
		},
		Data: make(map[string][]byte),
	}
	return secret
}

func NewK8SStoreFactory(cfg interface{}) (store.Store, error) {
	cfgStore, ok := cfg.(*StoreConfig)
	if !ok {
		return nil, fmt.Errorf("cfg is not *StoreConfig type")
	}
	// init the global singleton secret cluster store if it not exists; maybe put it in crane deployment is an option
	_, err := cfgStore.Client.CoreV1().Secrets(known.CraneSystemNamespace).Get(context.TODO(), CraneStoreClustersSecretName, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		secret := DefaultSecretStore()
		_, err := cfgStore.Client.CoreV1().Secrets(known.CraneSystemNamespace).Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			return nil, err
		}
	}

	once.Do(func() {
		k8sFactory = &datastore{client: cfgStore.Client}
	})
	return k8sFactory, nil
}

const StoreType = "secret"

func init() {
	store.RegisterFactory(StoreType, NewK8SStoreFactory)
}
