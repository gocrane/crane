package store

import (
	"context"
	"strings"

	utilrand "k8s.io/apimachinery/pkg/util/rand"
)

// Cluster define the craned endpoint and information about the cluster  the craned deployed in
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

type NamespaceList struct {
	TotalCount int      `json:"totalCount"`
	Items      []string `json:"items"`
}

// ClusterStore define the cluster store CURD interface
type ClusterStore interface {
	AddCluster(ctx context.Context, cluster *Cluster) error
	DeleteCluster(ctx context.Context, clusterid string) error
	UpdateCluster(ctx context.Context, cluster *Cluster) error
	GetCluster(ctx context.Context, clusterid string) (*Cluster, error)
	ListClusters(ctx context.Context) (*ClusterList, error)
	ListNamespaces(ctx context.Context, clusterid string) (*NamespaceList, error)
}

func (c Cluster) CraneUrlDuplicated(store map[string]*Cluster) bool {
	for id, cluster := range store {
		if id != c.Id {
			if cluster.CraneUrl == c.CraneUrl {
				return true
			}
		}
	}
	return false
}

func GenerateClusterName(base string) string {
	if strings.HasSuffix(base, "-") {
		return base + utilrand.String(8)
	}
	return base + "-" + utilrand.String(8)
}
