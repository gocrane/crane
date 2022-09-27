package clusters

import (
	"context"
	"fmt"
	"net/url"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	analysisapi "github.com/gocrane/api/analysis/v1alpha1"

	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/server/config"
	"github.com/gocrane/crane/pkg/server/ginwrapper"
	"github.com/gocrane/crane/pkg/server/service/cluster"
	"github.com/gocrane/crane/pkg/server/store"
)

const RecommendationRuleWorkloadsName = "workloads-rule"
const RecommendationRuleWorkloadsYAML = `
apiVersion: analysis.crane.io/v1alpha1
kind: RecommendationRule
metadata:
  name: workloads-rule
  labels:
    analysis.crane.io/recommendation-rule-preinstall: true
spec:
  runInterval: 24h                            # 每24h运行一次
  resourceSelectors:                          # 资源的信息
    - kind: Deployment
      apiVersion: apps/v1
    - kind: StatefulSet
      apiVersion: apps/v1
  namespaceSelector:
    any: true                                 # 扫描所有namespace
  recommenders:                               # 使用 Workload 的副本和资源推荐器
    - name: Replicas
    - name: Resource
`

const RecommendationRuleIdleNodeName = "idlenodes-rule"
const RecommendationRuleIdleNodeYAML = `
apiVersion: analysis.crane.io/v1alpha1
kind: RecommendationRule
metadata:
  name: idlenodes-rule
  labels:
    analysis.crane.io/recommendation-rule-preinstall: "true"
spec:
  runInterval: 24h                            # 每24h运行一次
  resourceSelectors:                          # 资源的信息
    - kind: Node
      apiVersion: v1
  namespaceSelector:
    any: true                                 # 扫描所有namespace
  recommenders:
    - name: IdleNode
`

type AddClustersRequest struct {
	Clusters []*store.Cluster `json:"clusters"`
}

type ClusterHandler struct {
	clusterSrv cluster.Service
	client     client.Client
}

// Note: cluster service is just used by front end to list clusters which added by user in frontend
func NewClusterHandler(srv cluster.Service, config *config.Config) *ClusterHandler {
	return &ClusterHandler{
		clusterSrv: srv,
		client:     config.Client,
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

	if len(r.Clusters) == 0 {
		ginwrapper.WriteResponse(c, fmt.Errorf("req is empty, check your input para"), nil)
		return
	}

	clustersMap, err := ch.getClusterMap()
	if err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}

	for _, cluster := range r.Clusters {
		if cluster.CraneUrl == "" || cluster.Name == "" {
			err := fmt.Errorf("cluster CraneUrl, Name field must not be empty")
			ginwrapper.WriteResponse(c, err, nil)
			return
		}

		if !IsUrl(cluster.CraneUrl) {
			err := fmt.Errorf("cluster CraneUrl %v is not valid url", cluster.CraneUrl)
			ginwrapper.WriteResponse(c, err, nil)
			return
		}

		if cluster.Id == "" {
			cluster.Id = store.GenerateClusterName("cls")
		}

		if _, ok := clustersMap[cluster.Id]; ok {
			err := fmt.Errorf("cluster id %v duplicated", cluster.Id)
			ginwrapper.WriteResponse(c, err, nil)
			return
		}

		if cluster.CraneUrlDuplicated(clustersMap) {
			err := fmt.Errorf("cluster CraneUlr %v duplicated", cluster.CraneUrl)
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

		if cluster.PreinstallRecommendation && err == nil {
			err := ch.upsertRecommendationRule(RecommendationRuleWorkloadsName, RecommendationRuleWorkloadsYAML)
			if err != nil {
				ginwrapper.WriteResponse(c, err, nil)
				return
			}

			err = ch.upsertRecommendationRule(RecommendationRuleIdleNodeName, RecommendationRuleIdleNodeYAML)
			if err != nil {
				ginwrapper.WriteResponse(c, err, nil)
				return
			}
		} else if err != nil {
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
	old.Discount = r.Discount

	clustersMap, err := ch.getClusterMap()
	if err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}

	if r.CraneUrlDuplicated(clustersMap) {
		err := fmt.Errorf("cluster CraneUlr %v duplicated", r.CraneUrl)
		ginwrapper.WriteResponse(c, err, nil)
		return
	}

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

func (ch *ClusterHandler) getClusterMap() (map[string]*store.Cluster, error) {
	clustersMap := map[string]*store.Cluster{}

	clusterList, err := ch.clusterSrv.ListClusters(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("list cluster err: %v", err)
	}

	for _, clu := range clusterList.Items {
		clustersMap[clu.Id] = clu
	}
	return clustersMap, nil
}

func (ch *ClusterHandler) upsertRecommendationRule(name string, yamlString string) error {
	key := types.NamespacedName{
		Namespace: known.CraneSystemNamespace,
		Name:      name,
	}
	var recommendationRule analysisapi.RecommendationRule
	err := ch.client.Get(context.TODO(), key, &recommendationRule)
	if err != nil {
		if errors.IsNotFound(err) {
			var workloadRecommendationRule analysisapi.RecommendationRule
			err = yaml.Unmarshal([]byte(yamlString), &workloadRecommendationRule)
			if err != nil {
				return err
			}
			err = ch.client.Create(context.TODO(), &workloadRecommendationRule)
			if err != nil {
				return err
			}
		} else {
			klog.Errorf("get preinstall recommendation failed: %v", err)
			return err
		}
	}

	return nil
}
