package recommender

import (
	"fmt"
	"sync"

	"k8s.io/klog/v2"

	analysisv1alph1 "github.com/gocrane/api/analysis/v1alpha1"
	"github.com/gocrane/crane/pkg/recommendation/recommender/apis"
)

type Factory func(apis.Recommender, analysisv1alph1.RecommendationRule) (Recommender, error)

// All registered Recommender providers.
var (
	providersMutex sync.Mutex
	providers      = make(map[string]Factory)
)

// RegisterRecommenderProvider registers a recommender.Factory by name.  This
// is expected to happen during app startup.
func RegisterRecommenderProvider(name string, recommender Factory) {
	providersMutex.Lock()
	defer providersMutex.Unlock()
	if _, found := providers[name]; found {
		klog.Fatalf("recommender provider %q was registered twice", name)
	}
	klog.V(1).Infof("Registered recommender provider %q", name)
	providers[name] = recommender
}

// GetRecommenderProvider creates an instance of the named Recommender provider, or nil if
// the name is unknown.  The error return is only used if the named provider
// was known but failed to initialize.
func GetRecommenderProvider(recommenderName string, recommender apis.Recommender, recommendationRule analysisv1alph1.RecommendationRule) (Recommender, error) {
	providersMutex.Lock()
	defer providersMutex.Unlock()
	f, found := providers[recommenderName]
	if !found {
		return nil, fmt.Errorf("unknown recommender name: %s", recommenderName)
	}
	return f(recommender, recommendationRule)
}
