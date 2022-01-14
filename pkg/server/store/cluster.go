package store

import "context"

// Cluster define some cluster and crane info deployed on the cluster
type Cluster struct {
	// Cluster Id must be unique
	Id string `json:"id"`
	// Cluster name
	Name string `json:"name"`
	// Crane server url in the cluster
	CraneUrl string `json:"craneUrl"`
	// Grafana url in the cluster
	GrafanaUrl string `json:"grafanaUrl"`
}

type ClusterList struct {
	TotalCount int32      `json:"totalCount"`
	Items      []*Cluster `json:"items"`
}

// ClusterStore define the cluster store CURD interface
type ClusterStore interface {
	AddCluster(ctx context.Context, cluster *Cluster, opts *CreateOptions) error
	DeleteCluster(ctx context.Context, clusterid string, opts *DeleteOptions) error
	UpdateCluster(ctx context.Context, cluster *Cluster, opts *UpdateOptions) error
	GetCluster(ctx context.Context, clusterid string, opts *GetOptions) (*Cluster, error)
	ListClusters(ctx context.Context, opts *ListOptions) (*ClusterList, error)
}
