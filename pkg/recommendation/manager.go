package recommendation

import (
	"fmt"
	"sync"

	"github.com/fsnotify/fsnotify"
	"k8s.io/klog/v2"

	analysisv1alph1 "github.com/gocrane/api/analysis/v1alpha1"
	"github.com/gocrane/crane/pkg/recommendation/config"
	"github.com/gocrane/crane/pkg/recommendation/framework"
	"github.com/gocrane/crane/pkg/recommendation/recommender"
	"github.com/gocrane/crane/pkg/recommendation/recommender/apis"
	_ "github.com/gocrane/crane/pkg/recommendation/recommender/hpa"
	_ "github.com/gocrane/crane/pkg/recommendation/recommender/idlenode"
	_ "github.com/gocrane/crane/pkg/recommendation/recommender/replicas"
	_ "github.com/gocrane/crane/pkg/recommendation/recommender/resource"
	_ "github.com/gocrane/crane/pkg/recommendation/recommender/service"
)

type RecommenderManager interface {
	// GetRecommender return a registered recommender
	GetRecommender(recommenderName string) (recommender.Recommender, error)
	// GetRecommenderWithRule return a registered recommender, its config merged with recommendationRule
	GetRecommenderWithRule(recommenderName string, recommendationRule analysisv1alph1.RecommendationRule) (recommender.Recommender, error)
}

func NewRecommenderManager(recommendationConfiguration string) RecommenderManager {
	m := &manager{
		recommendationConfiguration: recommendationConfiguration,
	}

	m.loadConfigFile() // nolint:errcheck

	go m.watchConfigFile()

	return m
}

type ResourceSpec struct {
	CPU    string
	Memory string
}

type ResourceSpecs []ResourceSpec

type manager struct {
	recommendationConfiguration string

	lock               sync.Mutex
	recommenderConfigs map[string]apis.Recommender
}

func (m *manager) GetRecommender(recommenderName string) (recommender.Recommender, error) {
	return m.GetRecommenderWithRule(recommenderName, analysisv1alph1.RecommendationRule{})
}

func (m *manager) GetRecommenderWithRule(recommenderName string, recommendationRule analysisv1alph1.RecommendationRule) (recommender.Recommender, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if recommenderConfig, ok := m.recommenderConfigs[recommenderName]; ok {
		return recommender.GetRecommenderProvider(recommenderName, recommenderConfig, recommendationRule)
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

	recommenderConfigs, err := config.GetRecommendersFromConfiguration(m.recommendationConfiguration)
	if err != nil {
		klog.ErrorS(err, "Failed to load recommendation config file", "file", m.recommendationConfiguration)
		return err
	}
	m.recommenderConfigs = recommenderConfigs
	klog.Info("Recommendation Config updated.")
	return nil
}

func Run(ctx *framework.RecommendationContext, recommender recommender.Recommender) error {
	klog.Infof("%s: start to run recommender %q.", ctx.String(), recommender.Name())

	// 1. Filter phase
	err := recommender.Filter(ctx)
	if err != nil {
		klog.Errorf("%s: recommender %q failed at filter phase: %v", ctx.String(), recommender.Name(), err)
		return err
	}

	// 2. PrePrepare phase
	err = recommender.CheckDataProviders(ctx)
	if err != nil {
		klog.Errorf("%s: recommender %q failed at prepare check data provider phase: %v", ctx.String(), recommender.Name(), err)
		return err
	}

	// 3. Prepare phase
	err = recommender.CollectData(ctx)
	if err != nil {
		klog.Errorf("%s: recommender %q failed at prepare collect data phase: %v", ctx.String(), recommender.Name(), err)
		return err
	}

	// 4. PostPrepare phase
	err = recommender.PostProcessing(ctx)
	if err != nil {
		klog.Errorf("%s: recommender %q failed at prepare data post processing phase: %v", ctx.String(), recommender.Name(), err)
		return err
	}

	// 5. PreRecommend phase
	err = recommender.PreRecommend(ctx)
	if err != nil {
		klog.Errorf("%s: recommender %q failed at pre commend phase: %v", ctx.String(), recommender.Name(), err)
		return err
	}

	// 6. Recommend phase
	err = recommender.Recommend(ctx)
	if err != nil {
		klog.Errorf("%s: recommender %q failed at recommend phase: %v", ctx.String(), recommender.Name(), err)
		return err
	}

	// 7. PostRecommend phase, add policy
	err = recommender.Policy(ctx)
	if err != nil {
		klog.Errorf("%s: recommender %q failed at recommend policy phase: %v", ctx.String(), recommender.Name(), err)
		return err
	}

	// 8. Observe phase
	err = recommender.Observe(ctx)
	if err != nil {
		klog.Errorf("%s: recommender %q failed at observe phase: %v", ctx.String(), recommender.Name(), err)
		return err
	}

	klog.Infof("%s: finish to run recommender %q.", ctx.String(), recommender.Name())
	return nil
}
