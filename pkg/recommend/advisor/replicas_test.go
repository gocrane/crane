package advisor

import (
	"math/rand"
	"testing"
	"time"

	"github.com/montanaflynn/stats"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/recommend/types"
)

func TestCheckFluctuation(t *testing.T) {
	a := &ReplicasAdvisor{
		Context: &types.Context{
			ConfigProperties: map[string]string{},
		},
	}

	var samples []common.Sample
	timeSample := time.Now()
	for i := 1; i < 1440; i++ {
		sample := common.Sample{
			Timestamp: timeSample.Unix(),
			Value:     float64(i),
		}
		timeSample = timeSample.Add(time.Duration(1) * time.Minute)
		samples = append(samples, sample)
	}

	tsList := []*common.TimeSeries{{Samples: samples}}

	tests := []struct {
		description string
		threshold   string
		expectError bool
	}{
		{
			description: "check fluctuation passed",
			threshold:   "3",
			expectError: false,
		},
		{
			description: "check fluctuation failed",
			threshold:   "100",
			expectError: true,
		},
	}

	medianMin, medianMax, _ := a.minMaxMedians(tsList)

	for _, test := range tests {
		a.Context.ConfigProperties["replicas.fluctuation-threshold"] = test.threshold
		err := a.checkFluctuation(medianMin, medianMax)
		if err != nil && !test.expectError {
			t.Errorf("Failed to checkFluctuation: %v", err)
		}
	}
}

func TestProposeMaxReplicas(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	a := &ReplicasAdvisor{
		Context: &types.Context{
			ConfigProperties: map[string]string{
				"replicas.max-replicas-factor": "3",
			},
			PodTemplate: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "podTemplateTest",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name: "container1",
						Resources: corev1.ResourceRequirements{
							Requests: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceCPU:    *resource.NewQuantity(1, resource.DecimalSI),
								corev1.ResourceMemory: *resource.NewQuantity(10, resource.DecimalSI),
							},
						},
					}, {
						Name: "container2",
						Resources: corev1.ResourceRequirements{
							Requests: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceCPU:    *resource.NewQuantity(1, resource.DecimalSI),
								corev1.ResourceMemory: *resource.NewQuantity(10, resource.DecimalSI),
							},
						},
					}},
				},
			},
		},
	}

	var cpuUsages []float64
	for i := 1; i < 1000; i++ {
		value := 1 + rand.Float64()*(20-1) // random float from [1,100]
		cpuUsages = append(cpuUsages, value)
	}

	percentileCpu, _ := stats.Percentile(cpuUsages, 95)

	tests := []struct {
		description           string
		targetUtilization     int32
		minReplicas           int32
		expertReplicasAtLeast int32
	}{
		{
			description:           "use targetUtilization==50",
			targetUtilization:     50,
			minReplicas:           20,
			expertReplicasAtLeast: 30,
		},
		{
			description:           "use targetUtilization==10",
			targetUtilization:     10,
			minReplicas:           20,
			expertReplicasAtLeast: 100,
		},
		{
			description:           "capping by minReplicas",
			targetUtilization:     50,
			minReplicas:           60,
			expertReplicasAtLeast: 60,
		},
	}

	for _, test := range tests {
		maxReplicas, err := a.proposeMaxReplicas(percentileCpu, test.targetUtilization, test.minReplicas)
		if err != nil {
			t.Errorf("Failed to checkFluctuation: %v", err)
		}
		if maxReplicas < test.expertReplicasAtLeast {
			t.Errorf("Failed to proposeMaxReplicas, expect at least %d actual %d.", test.expertReplicasAtLeast, maxReplicas)
		}
	}
}
