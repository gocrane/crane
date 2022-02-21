package server

import (
	"github.com/gocrane/crane/pkg/server/handler/clusters"
	"github.com/gocrane/crane/pkg/server/handler/dashboards"
)

func (s *apiServer) initRouter() {
	s.installHandler()
}

func (s *apiServer) installHandler() {

	clusterHandler := clusters.NewClusterHandler(s.clusterSrv)

	v1 := s.Group("/v1")
	{
		//dashboard panels
		if s.config.EnableGrafana {
			dbshandler := dashboards.NewDashboardHandler(s.dashboardSrv)
			dashboardsv1 := v1.Group("/dashboard")
			{
				dashboardsv1.GET("", dbshandler.List)
				dashboardsv1.GET("/panels", dbshandler.ListPanels)
			}
		}

		// prometheus

		// clusters
		clustersv1 := v1.Group("/cluster")
		{
			clustersv1.GET("", clusterHandler.ListClusters)
			clustersv1.POST("", clusterHandler.AddClusters)
			clustersv1.DELETE(":clusterid", clusterHandler.DeleteCluster)
			clustersv1.PUT(":clusterid", clusterHandler.UpdateCluster)
			clustersv1.GET(":clusterid", clusterHandler.GetCluster)

			// namespaces
			nsv1 := clustersv1.Group("/namespaces")
			{
				nsv1.GET(":clusterid", clusterHandler.ListNamespaces)
			}
		}
	}

}
