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
	clusterSrv       cluster.Service
	client           client.Client
	dashboardControl bool
}

// Note: cluster service is just used by front end to list clusters which added by user in frontend
func NewClusterHandler(srv cluster.Service, config *config.Config) *ClusterHandler {
	return &ClusterHandler{
		clusterSrv:       srv,
		client:           config.Client,
		dashboardControl: config.DashboardControl,
	}
}

// ListClusters list the clusters which has deployed crane server.
func (ch *ClusterHandler) ListClusters(c *gin.Context) {
	clusterList, err := ch.clusterSrv.ListClusters(context.TODO())
	if err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}
	// set dashboardControl based on serverConfig
	for index := range clusterList.Items {
		clusterList.Items[index].DashboardControl = ch.dashboardControl
	}
	ginwrapper.WriteResponse(c, nil, clusterList)
}

// AddClusters add the clusters which has deployed crane server. cluster info must has valid & accessible crane server url
func (ch *ClusterHandler) AddClusters(c *gin.Context) {
	var request AddClustersRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}

	if len(request.Clusters) == 0 {
		ginwrapper.WriteResponse(c, fmt.Errorf("no clusters provided"), nil)
		return
	}

	clustersMap, err := ch.getClusterMap()

	if err != nil {
		ginwrapper.WriteResponse(c, err, nil)
		return
	}

	var errors []error
	var addedClusters []*store.Cluster

	for _, cluster := range request.Clusters {
		if err := validateCluster(cluster); err != nil {
			errors = append(errors, err)
			continue
		}

		if _, ok := clustersMap[cluster.Id]; ok {
			errors = append(errors, fmt.Errorf("cluster id %v duplicated", cluster.Id))
			continue
		}

		if cluster.CraneUrlDuplicated(clustersMap) {
			errors = append(errors, fmt.Errorf("cluster CraneUrl %v duplicated", cluster.CraneUrl))
			continue
		}

		if cluster.Id == "" {
			cluster.Id = store.GenerateClusterName("cls")
		}

		// if cluster.Discount is not set,we should set the default value 100.
		if cluster.Discount == 0 {
			cluster.Discount = 100
		}

		clustersMap[cluster.Id] = cluster

		addedClusters = append(addedClusters, cluster)
	}

	if len(errors) > 0 {
		ginwrapper.WriteResponse(c, fmt.Errorf("encountered errors adding clusters: %v", errors), nil)
		return
	}

	for _, cluster := range addedClusters {
		if err := ch.clusterSrv.AddCluster(context.TODO(), cluster); err != nil {
			errors = append(errors, err)
			continue
		}

		if cluster.PreinstallRecommendation {
			if err := ch.upsertRecommendationRules(); err != nil {
				errors = append(errors, err)
				continue
			}
		}
	}

	if len(errors) > 0 {
		ginwrapper.WriteResponse(c, fmt.Errorf("encountered errors adding clusters: %v", errors), nil)
		return
	}

	ginwrapper.WriteResponse(c, nil, nil)
}

func validateCluster(cluster *store.Cluster) error {
	if cluster.CraneUrl == "" || cluster.Name == "" {
		return fmt.Errorf("cluster CraneUrl or Name field is empty")
	}

	if !IsUrl(cluster.CraneUrl) {
		return fmt.Errorf("cluster CraneUrl %v is not valid url", cluster.CraneUrl)
	}

	return nil
}

func (ch *ClusterHandler) upsertRecommendationRules() error {
	if err := ch.upsertRecommendationRule(RecommendationRuleWorkloadsName, RecommendationRuleWorkloadsYAML); err != nil {
		return err
	}

	if err := ch.upsertRecommendationRule(RecommendationRuleIdleNodeName, RecommendationRuleIdleNodeYAML); err != nil {
		return err
	}

	return nil
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

	// if r.Discount is not set,we should set the default value 100.
	if r.Discount == 0 {
		old.Discount = 100
	} else {
		old.Discount = r.Discount
	}

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
	// set dashboardControl based on serverConfig
	getCluster.DashboardControl = ch.dashboardControl
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
