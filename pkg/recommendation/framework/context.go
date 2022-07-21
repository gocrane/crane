package framework

import (
	"context"
	"github.com/gocrane/api/analysis/v1alpha1"
	"github.com/gocrane/crane/pkg/providers"

	"github.com/gocrane/crane/pkg/common"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RecommendationContext struct {
	context.Context
	// The kubernetes resource object of recommendation flow.
	Object client.Object
	// Time series data from data source.
	Values *common.TimeSeries
	// DataProviders contains data source of your recommendation flow.
	DataProviders map[providers.DataSourceType]providers.Interface
	// Recommendation store result of recommendation flow.
	Recommendation v1alpha1.Recommendation
	// When cancel channel accept signal indicates that the context has been canceled. The recommendation should stop executing as soon as possible.
	CancelCh <-chan struct{}
}

func NewRecommendationContext(object client.Object, dataProviders map[providers.DataSourceType]providers.Interface) RecommendationContext {
	return RecommendationContext{
		Object:        object,
		DataProviders: dataProviders,
	}
}

func (ctx *RecommendationContext) Canceled() bool {
	select {
	case <-ctx.CancelCh:
		return true
	default:
		return false
	}
}
