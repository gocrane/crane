package secret

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/server/store"
)

const (
	CraneStoreLabel              = "crane-store"
	CraneStoreClustersSecretName = "clusters-secret-store"
)

type clusters struct {
	k8s *datastore
	sync.Mutex
}

func addClusterToSecret(cluster *store.Cluster, secret *v1.Secret) error {
	data, err := json.Marshal(cluster)
	if err != nil {
		return err
	}
	secret.Data[cluster.Id] = data
	return nil
}

func updateClusterInSecret(cluster *store.Cluster, secret *v1.Secret) error {
	if _, ok := secret.Data[cluster.Id]; ok {
		newData, err := json.Marshal(cluster)
		if err != nil {
			return err
		}
		secret.Data[cluster.Id] = newData
		return nil
	} else {
		return errors.NewNotFound(schema.GroupResource{}, cluster.Id)
	}
}

func deleteClusterInSecret(clusterid string, secret *v1.Secret) error {
	delete(secret.Data, clusterid)
	return nil
}

func getClusterInSecret(clusterid string, secret *v1.Secret) (*store.Cluster, error) {
	var cluster store.Cluster
	clusterdata, ok := secret.Data[clusterid]
	if !ok {
		return nil, errors.NewNotFound(schema.GroupResource{}, clusterid)
	}
	err := json.Unmarshal(clusterdata, &cluster)
	if err != nil {
		return nil, err
	}
	return &cluster, nil
}

func listClusterInSecret(secret *v1.Secret) (*store.ClusterList, error) {
	clusters := make([]*store.Cluster, len(secret.Data))
	var idx int32
	for id, data := range secret.Data {
		clusters[idx] = &store.Cluster{}
		err := json.Unmarshal(data, clusters[idx])
		if err != nil {
			return nil, fmt.Errorf("failed to decode the cluster %v, err: %v", id, err)
		}
		idx++
	}
	sort.Slice(clusters, func(i, j int) bool {
		if clusters[i].Id < clusters[j].Id {
			return true
		} else {
			return false
		}
	})
	return &store.ClusterList{
		Items:      clusters,
		TotalCount: idx,
	}, nil
}

func (c *clusters) readSecretStore(ctx context.Context) (*v1.Secret, error) {
	secret, err := c.k8s.client.CoreV1().Secrets(known.CraneSystemNamespace).Get(ctx, CraneStoreClustersSecretName, metav1.GetOptions{})
	if err != nil {
		return secret, err
	}
	// make sure the secret.Data is not nil map
	if len(secret.Data) == 0 {
		secret.Data = make(map[string][]byte)
	}
	return secret, nil
}

func (c *clusters) writeSecretStore(ctx context.Context, secret *v1.Secret) (*v1.Secret, error) {
	return c.k8s.client.CoreV1().Secrets(known.CraneSystemNamespace).Update(ctx, secret, metav1.UpdateOptions{})
}

func (c *clusters) AddCluster(ctx context.Context, cluster *store.Cluster) error {
	c.Lock()
	defer c.Unlock()
	secret, err := c.readSecretStore(ctx)
	if err != nil {
		return err
	}
	err = addClusterToSecret(cluster, secret)
	if err != nil {
		return err
	}
	_, err = c.writeSecretStore(ctx, secret)
	return err
}

func (c *clusters) DeleteCluster(ctx context.Context, clusterid string) error {
	c.Lock()
	defer c.Unlock()

	secret, err := c.readSecretStore(ctx)
	if err != nil {
		return err
	}
	err = deleteClusterInSecret(clusterid, secret)
	if err != nil {
		return err
	}
	_, err = c.writeSecretStore(ctx, secret)
	return err
}

func (c *clusters) UpdateCluster(ctx context.Context, cluster *store.Cluster) error {
	c.Lock()
	defer c.Unlock()

	secret, err := c.readSecretStore(ctx)
	if err != nil {
		return err
	}
	err = updateClusterInSecret(cluster, secret)
	if err != nil {
		return err
	}
	_, err = c.writeSecretStore(ctx, secret)
	return err
}

func (c *clusters) GetCluster(ctx context.Context, clusterid string) (*store.Cluster, error) {
	c.Lock()
	defer c.Unlock()

	secret, err := c.readSecretStore(ctx)
	if err != nil {
		return nil, err
	}
	return getClusterInSecret(clusterid, secret)
}

func (c *clusters) ListClusters(ctx context.Context) (*store.ClusterList, error) {
	c.Lock()
	defer c.Unlock()

	secret, err := c.readSecretStore(ctx)
	if err != nil {
		return nil, err
	}
	return listClusterInSecret(secret)
}

func NewClusters(datastore *datastore) store.ClusterStore {
	return &clusters{k8s: datastore}
}
