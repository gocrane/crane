package framework

import (
	"context"

	"github.com/gocrane/api/analysis/v1alpha1"
	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/controller/analytics"
	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/prediction/config"
	predictormgr "github.com/gocrane/crane/pkg/predictor"
	"github.com/gocrane/crane/pkg/providers"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RecommendationContext struct {
	Context context.Context
	// The kubernetes resource object reference of recommendation flow.
	Identity analytics.ObjectIdentity
	// Time series data from data source.
	InputValues []*common.TimeSeries
	// Result series from prediction
	ResultValues []*common.TimeSeries
	// DataProviders contains data source of your recommendation flow.
	DataProviders map[providers.DataSourceType]providers.History
	// Recommendation store result of recommendation flow.

	Recommendation *v1alpha1.Recommendation
	// When cancel channel accept signal indicates that the context has been canceled. The recommendation should stop executing as soon as possible.
	// CancelCh <-chan struct{}
	// RecommendationRule for the context
	RecommendationRule v1alpha1.RecommendationRule
	// metrics namer for datasource provider
	MetricNamer metricnaming.MetricNamer
	// Algorithm Config
	AlgorithmConfig *config.Config
	// Manager of predict algorithm
	PredictorMgr predictormgr.Manager
	// Pod template
	PodTemplate *v1.PodTemplateSpec
	// Client
	Client client.Client
}

func NewRecommendationContext(context context.Context, identity analytics.ObjectIdentity, dataProviders map[providers.DataSourceType]providers.History, recommendation *v1alpha1.Recommendation, client client.Client) RecommendationContext {
	return RecommendationContext{
		Identity:       identity,
		Context:        context,
		DataProviders:  dataProviders,
		Recommendation: recommendation,
		Client:         client,
		//CancelCh:       context.Done(),
	}
}

//func (ctx *RecommendationContext) Canceled() bool {
//	select {
//	case <-ctx.CancelCh:
//		return true
//	default:
//		return false
//	}
//}
