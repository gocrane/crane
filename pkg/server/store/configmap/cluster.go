package configmap

import (
	"context"
	"encoding/json"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/server/store"
)

const CraneStoreLabel = "crane-store"

type clusters struct {
	k8s *datastore
}

// one configmap mapping to one cluster, each configmap identified by unique Name cluster.Id, and only label with crane-store in crane-system namespace can use.
func Cluster2ConfigMap(cluster *store.Cluster) (*v1.ConfigMap, error) {
	data, err := json.Marshal(cluster)
	if err != nil {
		return nil, err
	}
	clusterCM := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Id,
			Namespace: known.CraneSystemNamespace,
			Labels:    map[string]string{CraneStoreLabel: "true"},
		},
		BinaryData: map[string][]byte{
			"cluster": data,
		},
	}
	return clusterCM, nil
}

func ConfigMap2Cluster(clusterConfigmap *v1.ConfigMap) (*store.Cluster, error) {
	var cluster store.Cluster
	clusterdata := clusterConfigmap.BinaryData["cluster"]
	err := json.Unmarshal(clusterdata, &cluster)
	if err != nil {
		return nil, err
	}
	return &cluster, nil
}

func (c clusters) AddCluster(ctx context.Context, cluster *store.Cluster, opts *store.CreateOptions) error {
	cm, err := Cluster2ConfigMap(cluster)
	if err != nil {
		return err
	}
	_, err = c.k8s.client.CoreV1().ConfigMaps(known.CraneSystemNamespace).Create(ctx, cm, metav1.CreateOptions{})
	return err
}

func (c clusters) DeleteCluster(ctx context.Context, clusterid string, opts *store.DeleteOptions) error {
	return c.k8s.client.CoreV1().ConfigMaps(known.CraneSystemNamespace).Delete(ctx, clusterid, metav1.DeleteOptions{})
}

func (c clusters) UpdateCluster(ctx context.Context, cluster *store.Cluster, opts *store.UpdateOptions) error {
	cm, err := Cluster2ConfigMap(cluster)
	if err != nil {
		return err
	}
	_, err = c.k8s.client.CoreV1().ConfigMaps(known.CraneSystemNamespace).Update(ctx, cm, metav1.UpdateOptions{})
	return err
}

func (c clusters) GetCluster(ctx context.Context, clusterid string, opts *store.GetOptions) (*store.Cluster, error) {
	cm, err := c.k8s.client.CoreV1().ConfigMaps(known.CraneSystemNamespace).Get(ctx, clusterid, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return ConfigMap2Cluster(cm)
}

func (c clusters) ListClusters(ctx context.Context, opts *store.ListOptions) (*store.ClusterList, error) {
	selector := &metav1.LabelSelector{
		MatchLabels: map[string]string{CraneStoreLabel: "true"},
	}
	ls, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return nil, err
	}

	cms, err := c.k8s.client.CoreV1().ConfigMaps(known.CraneSystemNamespace).List(ctx, metav1.ListOptions{LabelSelector: ls.String()})
	if err != nil {
		return nil, err
	}
	var cnt int32
	clusters := make([]*store.Cluster, 0)
	for _, cm := range cms.Items {
		cluster, err := ConfigMap2Cluster(&cm)
		if err != nil {
			return nil, err
		}
		cnt++
		clusters = append(clusters, cluster)
	}
	return &store.ClusterList{TotalCount: cnt, Items: clusters}, nil
}

func NewClusters(datastore *datastore) store.ClusterStore {
	return &clusters{k8s: datastore}
}
