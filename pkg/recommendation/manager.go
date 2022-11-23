package recommendation

import (
	"fmt"
	"sync"

	"github.com/fsnotify/fsnotify"
	"k8s.io/klog/v2"

	"github.com/gocrane/crane/pkg/oom"
	"github.com/gocrane/crane/pkg/providers"
	"github.com/gocrane/crane/pkg/recommendation/config"
	"github.com/gocrane/crane/pkg/recommendation/framework"
	"github.com/gocrane/crane/pkg/recommendation/recommender"
	"github.com/gocrane/crane/pkg/recommendation/recommender/apis"
	"github.com/gocrane/crane/pkg/recommendation/recommender/hpa"
	"github.com/gocrane/crane/pkg/recommendation/recommender/idlenode"
	"github.com/gocrane/crane/pkg/recommendation/recommender/replicas"
	"github.com/gocrane/crane/pkg/recommendation/recommender/resource"
)

type RecommenderManager interface {
	// GetRecommender return a registered recommender
	GetRecommender(recommenderName string) (recommender.Recommender, error)
}

func NewRecommenderManager(recommendationConfiguration string, oomRecorder oom.Recorder, realtimeDataSources map[providers.DataSourceType]providers.RealTime, historyDataSources map[providers.DataSourceType]providers.History) RecommenderManager {
	m := &manager{
		recommendationConfiguration: recommendationConfiguration,
		oomRecorder:                 oomRecorder,
	}
	go m.watchConfigFile()

	return m
}

type manager struct {
	recommendationConfiguration string

	lock               sync.Mutex
	recommenderConfigs []apis.Recommender
	oomRecorder        oom.Recorder
}

func (m *manager) GetRecommender(recommenderName string) (recommender.Recommender, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, r := range m.recommenderConfigs {
		if r.Name == recommenderName {
			switch recommenderName {
			case recommender.ReplicasRecommender:
				return replicas.NewReplicasRecommender(r)
			case recommender.HPARecommender:
				return hpa.NewHPARecommender(r)
			case recommender.ResourceRecommender:
				return resource.NewResourceRecommender(r, m.oomRecorder)
			case recommender.IdleNodeRecommender:
				return idlenode.NewIdleNodeRecommender(r)
			default:
				return nil, fmt.Errorf("unknown recommender name: %s", recommenderName)
			}
		}
	}
	return nil, fmt.Errorf("unknown recommender name: %s", recommenderName)
}

func (m *manager) watchConfigFile() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		klog.Error(err)
		return
	}
	defer watcher.Close()

	err = watcher.Add(m.recommendationConfiguration)
	if err != nil {
		klog.ErrorS(err, "Failed to watch", "file", m.recommendationConfiguration)
		return
	}
	klog.Infof("Start watching %s for update.", m.recommendationConfiguration)

	for {
		select {
		case event, ok := <-watcher.Events:
			klog.Infof("Watched an event: %v", event)
			if !ok {
				return
			}
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				err = watcher.Add(m.recommendationConfiguration)
				if err != nil {
					klog.ErrorS(err, "Failed to watch.", "file", m.recommendationConfiguration)
					continue
				}
				klog.Infof("Config file %s removed. Reload it.", event.Name)
				if err = m.loadConfigFile(); err != nil {
					klog.ErrorS(err, "Failed to load config set file.")
				}
			} else if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				klog.Infof("Config file %s modified. Reload it.", event.Name)
				if err = m.loadConfigFile(); err != nil {
					klog.ErrorS(err, "Failed to load config set file.")
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			klog.Error(err)
		}
	}
}

func (m *manager) loadConfigFile() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	apiRecommenders, err := config.GetRecommendersFromConfiguration(m.recommendationConfiguration)
	if err != nil {
		klog.ErrorS(err, "Failed to load recommendation config file", "file", m.recommendationConfiguration)
		return err
	}
	m.recommenderConfigs = apiRecommenders
	klog.Info("Recommendation Config updated.")
	return nil
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
