package clusters

import (
	"context"
	"fmt"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/gocrane/crane/pkg/server/ginwrapper"
	"github.com/gocrane/crane/pkg/server/service/cluster"
	"github.com/gocrane/crane/pkg/server/store"
)

type AddClustersRequest struct {
	Clusters []*store.Cluster `json:"clusters"`
}

type ClusterHandler struct {
	clusterSrv cluster.Service
}

// Note: cluster service is just used by front end to list clusters which added by user in frontend
func NewClusterHandler(srv cluster.Service) *ClusterHandler {
	return &ClusterHandler{
		clusterSrv: srv,
	}
}

// ListClusters list the clusters which has deployed crane server.
func (ch *ClusterHandler) ListClusters(c *gin.Context) {

	clusterList, err := ch.clusterSrv.ListClusters(context.TODO())
	if err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}
	ginwrapper.WriteResponse(c, nil, clusterList)
}

// AddClusters add the clusters which has deployed crane server. cluster info must has valid & accessible crane server url
func (ch *ClusterHandler) AddClusters(c *gin.Context) {
	var r AddClustersRequest
	if err := c.ShouldBindJSON(&r); err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}

	clustersMap := map[string]*store.Cluster{}

	for _, cluster := range r.Clusters {
		if cluster.CraneUrl == "" || cluster.Id == "" || cluster.Name == "" {
			err := fmt.Errorf("cluster CraneUrl, Id, Name field must not be empty")
			ginwrapper.WriteResponse(c, err, nil)
			return
		}
		if !IsUrl(cluster.CraneUrl) {
			err := fmt.Errorf("cluster CraneUrl %v is not valid url", cluster.CraneUrl)
			ginwrapper.WriteResponse(c, err, nil)
			return
		}
		if _, ok := clustersMap[cluster.Id]; ok {
			err := fmt.Errorf("cluster id %v duplicated", cluster.Id)
			ginwrapper.WriteResponse(c, err, nil)
			return
		}
		clustersMap[cluster.Id] = cluster
	}

	for _, cluster := range r.Clusters {
		err := ch.clusterSrv.AddCluster(context.TODO(), cluster)
		if err != nil {
			ginwrapper.WriteResponse(c, err, nil)
			return
		}
	}
	ginwrapper.WriteResponse(c, nil, nil)
}

// UpdateCluster the clusters crane info
func (ch *ClusterHandler) UpdateCluster(c *gin.Context) {
	var r store.Cluster
	if err := c.ShouldBindJSON(&r); err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}
	old, err := ch.clusterSrv.GetCluster(c, c.Param("clusterid"))
	if err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}
	old.Name = r.Name
	old.GrafanaUrl = r.GrafanaUrl
	old.CraneUrl = r.CraneUrl

	err = ch.clusterSrv.UpdateCluster(context.TODO(), old)
	if err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}
	ginwrapper.WriteResponse(c, nil, nil)

}

// DeleteCluster del the clusters
func (ch *ClusterHandler) DeleteCluster(c *gin.Context) {

	err := ch.clusterSrv.DeleteCluster(context.TODO(), c.Param("clusterid"))
	if err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}
	ginwrapper.WriteResponse(c, nil, nil)
}

// GetCluster return the cluster which has been added by front end.
func (ch *ClusterHandler) GetCluster(c *gin.Context) {
	getCluster, err := ch.clusterSrv.GetCluster(context.TODO(), c.Param("clusterid"))
	if err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}
	ginwrapper.WriteResponse(c, nil, getCluster)
}

// GetNamespaces return namespaces in specified cluster
func (ch *ClusterHandler) ListNamespaces(c *gin.Context) {
	getNamespaces, err := ch.clusterSrv.ListNamespaces(context.TODO(), c.Param("clusterid"))
	if err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}
	ginwrapper.WriteResponse(c, nil, getNamespaces)
}

func IsUrl(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}
