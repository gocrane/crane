package server

import (
	"github.com/gocrane/crane/pkg/server/handler/clusters"
	"github.com/gocrane/crane/pkg/server/handler/dashboards"
	"github.com/gocrane/crane/pkg/server/handler/prediction"
	"github.com/gocrane/crane/pkg/server/handler/prometheus"
	"github.com/gocrane/crane/pkg/server/handler/recommendation"
)

func (s *apiServer) initRouter() {
	clusterHandler := clusters.NewClusterHandler(s.clusterSrv)
	recommendationHandler := recommendation.NewRecommendationHandler(s.config)
	prometheusHandler := prometheus.NewPrometheusAPIHandler(s.config)

	v1 := s.Group("/api/v1")
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
		}

		// namespaces
		nsv1 := v1.Group("/namespaces")
		{
			nsv1.GET(":clusterid", clusterHandler.ListNamespaces)
		}

		// recommendations
		recommendv1 := v1.Group("/recommendation")
		{
			recommendv1.GET("", recommendationHandler.ListRecommendations)
		}

		// recommendationRules
		recommendrulev1 := v1.Group("/recommendationRule")
		{
			recommendrulev1.GET("", recommendationHandler.ListRecommendationRules)
			recommendrulev1.POST("", recommendationHandler.CreateRecommendationRule)
			recommendrulev1.PUT(":recommendationRuleName", recommendationHandler.UpdateRecommendationRule)
			recommendrulev1.DELETE(":recommendationRuleName", recommendationHandler.DeleteRecommendationRule)
		}

		// prometheus API
		prometheusapiv1 := v1.Group("/prometheus")
		{
			prometheusapiv1.GET("query", prometheusHandler.Query)
			prometheusapiv1.GET("query_range", prometheusHandler.RangeQuery)
		}
	}

	debugHandler := prediction.NewDebugHandler(s.config)
	debug := s.Group("/api/prediction/debug")
	{
		debug.GET(":namespace/:tsp", debugHandler.Display)
	}

}
