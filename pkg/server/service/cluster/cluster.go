package cluster

import (
	"context"

	"github.com/gocrane/crane/pkg/server/store"
)

type Service interface {
	AddCluster(ctx context.Context, cluster *store.Cluster) error
	DeleteCluster(ctx context.Context, clusterid string) error
	UpdateCluster(ctx context.Context, cluster *store.Cluster) error
	GetCluster(ctx context.Context, clusterid string) (*store.Cluster, error)
	ListClusters(ctx context.Context) (*store.ClusterList, error)
	ListNamespaces(ctx context.Context, clusterid string) (*store.NamespaceList, error)
}

type clusterService struct {
	datastore store.Store
}

func NewService(datastore store.Store) *clusterService {
	return &clusterService{datastore: datastore}
}

func (s *clusterService) AddCluster(ctx context.Context, cluster *store.Cluster) error {
	return s.datastore.Clusters().AddCluster(ctx, cluster)
}

func (s *clusterService) DeleteCluster(ctx context.Context, clusterid string) error {
	return s.datastore.Clusters().DeleteCluster(ctx, clusterid)
}

func (s *clusterService) UpdateCluster(ctx context.Context, cluster *store.Cluster) error {
	return s.datastore.Clusters().UpdateCluster(ctx, cluster)
}

func (s *clusterService) GetCluster(ctx context.Context, clusterid string) (*store.Cluster, error) {
	return s.datastore.Clusters().GetCluster(ctx, clusterid)
}

func (s *clusterService) ListClusters(ctx context.Context) (*store.ClusterList, error) {
	return s.datastore.Clusters().ListClusters(ctx)
}

func (s *clusterService) ListNamespaces(ctx context.Context, clusterid string) (*store.NamespaceList, error) {
	return s.datastore.Clusters().ListNamespaces(ctx, clusterid)
}
