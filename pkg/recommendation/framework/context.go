package framework

import (
	"context"
	"github.com/gocrane/api/analysis/v1alpha1"
	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/controller/analytics"
	"github.com/gocrane/crane/pkg/providers"
)

type RecommendationContext struct {
	Context context.Context
	// The kubernetes resource object reference of recommendation flow.
	Identity analytics.ObjectIdentity
	// Time series data from data source.
	Values *common.TimeSeries
	// DataProviders contains data source of your recommendation flow.
	DataProviders map[providers.DataSourceType]providers.Interface
	// Recommendation store result of recommendation flow.
	Recommendation v1alpha1.Recommendation
	// When cancel channel accept signal indicates that the context has been canceled. The recommendation should stop executing as soon as possible.
	CancelCh <-chan struct{}
}

func NewRecommendationContext(context context.Context, identity analytics.ObjectIdentity, dataProviders map[providers.DataSourceType]providers.Interface) RecommendationContext {
	return RecommendationContext{
		Identity:      identity,
		Context:       context,
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
