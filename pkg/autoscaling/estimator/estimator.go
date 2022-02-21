package estimator

import (
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"

	"github.com/gocrane/crane/pkg/oom"
)

// ResourceEstimatorManager controls how to get or delete estimators for EffectiveVPA
type ResourceEstimatorManager interface {

	// GetEstimators get estimator instances based on EffectiveVPA spec
	GetEstimators(evpa *autoscalingapi.EffectiveVerticalPodAutoscaler) []ResourceEstimatorInstance

	// DeleteEstimators release estimator resources based on EffectiveVPA spec
	DeleteEstimators(evpa *autoscalingapi.EffectiveVerticalPodAutoscaler)
}

type estimatorManager struct {
	mu sync.Mutex

	// estimatorMap save build-in estimators, type -> estimator
	estimatorMap map[string]ResourceEstimator
}

func NewResourceEstimatorManager(client client.Client, oomRecorder oom.Recorder) ResourceEstimatorManager {
	resourceEstimatorManager := &estimatorManager{
		estimatorMap: make(map[string]ResourceEstimator),
	}
	resourceEstimatorManager.buildEstimators(client, oomRecorder)
	return resourceEstimatorManager
}

func (m *estimatorManager) buildEstimators(client client.Client, oomRecorder oom.Recorder) {
	proportionalEstimator := &ProportionalResourceEstimator{
		Client: client,
	}
	m.registerEstimator("Proportional", proportionalEstimator)
	oomEstimator := &OOMResourceEstimator{
		OOMRecorder: oomRecorder,
	}
	m.registerEstimator("OOM", oomEstimator)
}

func (m *estimatorManager) GetEstimators(evpa *autoscalingapi.EffectiveVerticalPodAutoscaler) []ResourceEstimatorInstance {
	m.mu.Lock()
	defer m.mu.Unlock()

	var resourceEstimatorInstances []ResourceEstimatorInstance

	for _, estimatorSpec := range evpa.Spec.ResourceEstimators {
		estimator := m.estimatorMap[estimatorSpec.Type]
		var estimatorInstance resourceEstimatorInstance
		if estimator == nil {
			// can't found in estimatorMap, create a external estimator and register it
			estimatorInstance = resourceEstimatorInstance{
				ResourceEstimator: NewExternalResourceEstimator(estimatorSpec),
				Spec:              estimatorSpec,
			}
			m.registerEstimator(estimatorSpec.Type, estimatorInstance.ResourceEstimator)
		} else {
			estimatorInstance = resourceEstimatorInstance{
				ResourceEstimator: estimator,
				Spec:              estimatorSpec,
			}
		}

		resourceEstimatorInstances = append(resourceEstimatorInstances, estimatorInstance)
	}

	return resourceEstimatorInstances
}

func (m *estimatorManager) DeleteEstimators(evpa *autoscalingapi.EffectiveVerticalPodAutoscaler) {
	for _, estimatorSpec := range evpa.Spec.ResourceEstimators {
		estimator := m.estimatorMap[estimatorSpec.Type]
		if estimator == nil {
			klog.Warning("Delete estimators failed, type %s not found. ", estimatorSpec.Type)
			return
		}
		estimator.DeleteEstimation(evpa)
	}
}

// registerEstimator register a estimator in estimatorMap
func (m *estimatorManager) registerEstimator(estimatorType string, estimator ResourceEstimator) {
	if _, exist := m.estimatorMap[estimatorType]; !exist {
		m.estimatorMap[estimatorType] = estimator
	}
}

// ResourceEstimator defines the implement for build-in estimator
type ResourceEstimator interface {

	// GetResourceEstimation get estimated resource result for an EffectiveVPA and related configs
	GetResourceEstimation(evpa *autoscalingapi.EffectiveVerticalPodAutoscaler, config map[string]string, containerName string, currRes *corev1.ResourceRequirements) (corev1.ResourceList, error)

	// DeleteEstimation delete related resource from an EffectiveVPA
	DeleteEstimation(evpa *autoscalingapi.EffectiveVerticalPodAutoscaler)
}

// ResourceEstimatorInstance is the instance that used for container scaling
type ResourceEstimatorInstance interface {
	ResourceEstimator

	// GetSpec return the spec for this instance
	GetSpec() autoscalingapi.ResourceEstimator
}

type resourceEstimatorInstance struct {
	ResourceEstimator
	Spec autoscalingapi.ResourceEstimator
}

func (e resourceEstimatorInstance) GetSpec() autoscalingapi.ResourceEstimator {
	return e.Spec
}
