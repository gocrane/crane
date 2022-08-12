package recommendation

import (
	"sync"

	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/providers"
	"github.com/gocrane/crane/pkg/recommendation/framework"
	"github.com/gocrane/crane/pkg/recommendation/recommender"
)

type RecommenderManager interface {
	// GetRecommender return a registered recommender
	GetRecommender(recommenderName string) recommender.Recommender
}

func NewRecommenderManager(recommenders map[string]recommender.Recommender, realtimeDataSources map[providers.DataSourceType]providers.RealTime, historyDataSources map[providers.DataSourceType]providers.History) RecommenderManager {
	return &manager{
		recommenders:        recommenders,
		realtimeDataSources: realtimeDataSources,
		historyDataSources:  historyDataSources,
	}
}

type manager struct {
	lock                sync.Mutex
	recommenders        map[string]recommender.Recommender
	realtimeDataSources map[providers.DataSourceType]providers.RealTime
	historyDataSources  map[providers.DataSourceType]providers.History
}

func (m *manager) GetRecommender(recommenderName string) recommender.Recommender {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.recommenders[recommenderName]
}

func Run(ctx *framework.RecommendationContext, recommender recommender.Recommender) error {
	//// If context is canceled, directly return.
	//if ctx.Canceled() {
	//	klog.Infof("Recommender %q has been cancelled...", recommender.Name())
	//	return nil
	//}

	// 1. Filter phase
	err := recommender.Filter(ctx)
	if err != nil {
		klog.Errorf("recommender %q failed at filter phase!", recommender.Name())
		return err
	}

	// 2. PrePrepare phase
	err = recommender.CheckDataProviders(ctx)
	if err != nil {
		klog.Errorf("recommender %q failed at prepare check data provider phase!", recommender.Name())
		return err
	}

	// 3. Prepare phase
	err = recommender.CollectData(ctx)
	if err != nil {
		klog.Errorf("recommender %q failed at prepare collect data phase!", recommender.Name())
		return err
	}

	// 4. PostPrepare phase
	err = recommender.PostProcessing(ctx)
	if err != nil {
		klog.Errorf("recommender %q failed at prepare data post processing phase!", recommender.Name())
		return err
	}

	// 5. PreRecommend phase
	err = recommender.PreRecommend(ctx)
	if err != nil {
		klog.Errorf("recommender %q failed at pre commend phase!", recommender.Name())
		return err
	}

	// 6. Recommend phase
	err = recommender.Recommend(ctx)
	if err != nil {
		klog.Errorf("recommender %q failed at recommend phase!", recommender.Name())
		return err
	}

	// 7. PostRecommend phase, add policy
	err = recommender.Policy(ctx)
	if err != nil {
		klog.Errorf("recommender %q failed at recommend policy phase!", recommender.Name())
		return err
	}

	// 8. Observe phase
	err = recommender.Observe(ctx)
	if err != nil {
		klog.Errorf("recommender %q failed at observe phase!", recommender.Name())
		return err
	}
	return nil
}
