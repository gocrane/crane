package evpa

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"

	"github.com/gocrane/crane/pkg/autoscaling/estimator"
)

type TestResourceEstimatorInstance struct {
	estimator.ResourceEstimator
	Spec autoscalingapi.ResourceEstimator
}

func (e TestResourceEstimatorInstance) GetSpec() autoscalingapi.ResourceEstimator {
	return e.Spec
}

func TestRankEstimators(t *testing.T) {
	resourceEstimators := []estimator.ResourceEstimatorInstance{
		&TestResourceEstimatorInstance{
			ResourceEstimator: &estimator.ProportionalResourceEstimator{},
			Spec: autoscalingapi.ResourceEstimator{
				Type:     "type1",
				Priority: 10,
			},
		},
		&TestResourceEstimatorInstance{
			ResourceEstimator: &estimator.ProportionalResourceEstimator{},
			Spec: autoscalingapi.ResourceEstimator{
				Type:     "type2",
				Priority: 10,
			},
		},
		&TestResourceEstimatorInstance{
			ResourceEstimator: &estimator.ProportionalResourceEstimator{},
			Spec: autoscalingapi.ResourceEstimator{
				Type:     "type3",
				Priority: 10,
			},
		},
		&TestResourceEstimatorInstance{
			ResourceEstimator: &estimator.ProportionalResourceEstimator{},
			Spec: autoscalingapi.ResourceEstimator{
				Type:     "type4",
				Priority: 20,
			},
		},
		&TestResourceEstimatorInstance{
			ResourceEstimator: &estimator.ProportionalResourceEstimator{},
			Spec: autoscalingapi.ResourceEstimator{
				Type:     "type5",
				Priority: 20,
			},
		},
	}

	ranked := RankEstimators(resourceEstimators)
	assert.True(t, len(ranked) == 2)
	assert.True(t, len(ranked[0].Estimators) == 3)
	assert.True(t, len(ranked[1].Estimators) == 2)
}

func TestCalculateResourceByValue(t *testing.T) {
	resourceEstimated := v1.ResourceList{
		v1.ResourceCPU:    *resource.NewQuantity(1, resource.DecimalSI),
		v1.ResourceMemory: *resource.NewQuantity(1024, resource.BinarySI),
	}

	tests := []struct {
		description string
		source      v1.ResourceList
		expect      v1.ResourceList
	}{
		{
			description: "calculate when resource empty",
			source:      v1.ResourceList{},
			expect: v1.ResourceList{
				v1.ResourceCPU:    *resource.NewQuantity(1, resource.DecimalSI),
				v1.ResourceMemory: *resource.NewQuantity(1024, resource.BinarySI),
			},
		},
		{
			description: "calculate cpu source",
			source: v1.ResourceList{
				v1.ResourceCPU: *resource.NewMilliQuantity(100, resource.DecimalSI),
			},
			expect: v1.ResourceList{
				v1.ResourceCPU:    *resource.NewQuantity(1, resource.DecimalSI),
				v1.ResourceMemory: *resource.NewQuantity(1024, resource.BinarySI),
			},
		},
		{
			description: "source is larger than estimated",
			source: v1.ResourceList{
				v1.ResourceCPU:    *resource.NewQuantity(2, resource.DecimalSI),
				v1.ResourceMemory: *resource.NewQuantity(2048, resource.BinarySI),
			},
			expect: v1.ResourceList{
				v1.ResourceCPU:    *resource.NewQuantity(2, resource.DecimalSI),
				v1.ResourceMemory: *resource.NewQuantity(2048, resource.BinarySI),
			},
		},
		{
			description: "estimated is larger than source",
			source: v1.ResourceList{
				v1.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
				v1.ResourceMemory: *resource.NewQuantity(512, resource.BinarySI),
			},
			expect: v1.ResourceList{
				v1.ResourceCPU:    *resource.NewQuantity(1, resource.DecimalSI),
				v1.ResourceMemory: *resource.NewQuantity(1024, resource.BinarySI),
			},
		},
	}

	for _, test := range tests {
		CalculateResourceByValue(test.source, resourceEstimated)
		if !test.source.Cpu().Equal(*test.expect.Cpu()) || !test.source.Memory().Equal(*test.expect.Memory()) {
			t.Errorf("expect result %v actual result %v", test.expect, test.source)
		}
	}
}

func TestCalculateResourceByPriority(t *testing.T) {
	resourceEstimated := v1.ResourceList{
		v1.ResourceCPU:    *resource.NewQuantity(1, resource.DecimalSI),
		v1.ResourceMemory: *resource.NewQuantity(1024, resource.BinarySI),
	}

	tests := []struct {
		description string
		source      v1.ResourceList
		expect      v1.ResourceList
	}{
		{
			description: "calculate when resource empty",
			source:      v1.ResourceList{},
			expect: v1.ResourceList{
				v1.ResourceCPU:    *resource.NewQuantity(1, resource.DecimalSI),
				v1.ResourceMemory: *resource.NewQuantity(1024, resource.BinarySI),
			},
		},
		{
			description: "calculate cpu source",
			source: v1.ResourceList{
				v1.ResourceCPU: *resource.NewMilliQuantity(100, resource.DecimalSI),
			},
			expect: v1.ResourceList{
				v1.ResourceCPU:    *resource.NewQuantity(1, resource.DecimalSI),
				v1.ResourceMemory: *resource.NewQuantity(1024, resource.BinarySI),
			},
		},
		{
			description: "source is larger than estimated",
			source: v1.ResourceList{
				v1.ResourceCPU:    *resource.NewQuantity(2, resource.DecimalSI),
				v1.ResourceMemory: *resource.NewQuantity(2048, resource.BinarySI),
			},
			expect: v1.ResourceList{
				v1.ResourceCPU:    *resource.NewQuantity(1, resource.DecimalSI),
				v1.ResourceMemory: *resource.NewQuantity(1024, resource.BinarySI),
			},
		},
		{
			description: "estimated is larger than source",
			source: v1.ResourceList{
				v1.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
				v1.ResourceMemory: *resource.NewQuantity(512, resource.BinarySI),
			},
			expect: v1.ResourceList{
				v1.ResourceCPU:    *resource.NewQuantity(1, resource.DecimalSI),
				v1.ResourceMemory: *resource.NewQuantity(1024, resource.BinarySI),
			},
		},
	}

	for _, test := range tests {
		resource := CalculateResourceByPriority([]v1.ResourceList{test.source, resourceEstimated})
		if !resource.Cpu().Equal(*test.expect.Cpu()) || !resource.Memory().Equal(*test.expect.Memory()) {
			t.Errorf("expect result %v actual result %v", test.expect, resource)
		}
	}
}
