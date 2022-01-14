package cluster

import (
	"context"

	"github.com/gocrane/crane/pkg/server/store"
)

type ClusterSrv interface {
	AddCluster(ctx context.Context, cluster *store.Cluster, opts *store.CreateOptions) error
	DeleteCluster(ctx context.Context, clusterid string, opts *store.DeleteOptions) error
	UpdateCluster(ctx context.Context, cluster *store.Cluster, opts *store.UpdateOptions) error
	GetCluster(ctx context.Context, clusterid string, opts *store.GetOptions) (*store.Cluster, error)
	ListClusters(ctx context.Context, opts *store.ListOptions) (*store.ClusterList, error)
}

type manager struct {
	datastore store.Factory
}

func NewManager(datastore store.Factory) *manager {
	return &manager{datastore: datastore}
}

func (m *manager) AddCluster(ctx context.Context, cluster *store.Cluster, opts *store.CreateOptions) error {
	return m.datastore.Clusters().AddCluster(ctx, cluster, opts)
}

func (m *manager) DeleteCluster(ctx context.Context, clusterid string, opts *store.DeleteOptions) error {
	return m.datastore.Clusters().DeleteCluster(ctx, clusterid, opts)
}

func (m *manager) UpdateCluster(ctx context.Context, cluster *store.Cluster, opts *store.UpdateOptions) error {
	return m.datastore.Clusters().UpdateCluster(ctx, cluster, opts)
}

func (m *manager) GetCluster(ctx context.Context, clusterid string, opts *store.GetOptions) (*store.Cluster, error) {
	return m.datastore.Clusters().GetCluster(ctx, clusterid, opts)
}

func (m *manager) ListClusters(ctx context.Context, opts *store.ListOptions) (*store.ClusterList, error) {
	return m.datastore.Clusters().ListClusters(ctx, opts)
}
