package sort

import (
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/gocrane/crane/pkg/ensurance/executor/podinfo"
)

func TestCpuUsageSorter(t *testing.T) {
	now := metav1.NewTime(time.Unix(1000, 0).UTC())
	later := metav1.NewTime(time.Unix(2000, 0).UTC())
	// orderedBy(UseElasticCPU, ComparePodQOSClass, ComparePriority, CompareCPUUsage, CompareElasticCPU, CompareRunningTime).Sort(pods)
	pods := []podinfo.PodContext{
		{
			Key:        types.NamespacedName{Name: "elastic-cpu-2"},
			ElasticCPU: 2,
			QOSClass:   v1.PodQOSBestEffort,
		},
		{
			Key:        types.NamespacedName{Name: "elastic-cpu-4"},
			ElasticCPU: 4,
			QOSClass:   v1.PodQOSBestEffort,
		},
		{
			Key:         types.NamespacedName{Name: "cpu-1"},
			PodCPUUsage: 1,
			QOSClass:    v1.PodQOSGuaranteed,
		},
		{
			Key:         types.NamespacedName{Name: "cpu-2"},
			PodCPUUsage: 2,
			QOSClass:    v1.PodQOSBurstable,
		},
		{
			Key:         types.NamespacedName{Name: "guarantee-1"},
			PodCPUUsage: 1,
			QOSClass:    v1.PodQOSGuaranteed,
		},
		{
			Key:         types.NamespacedName{Name: "burstable-2"},
			PodCPUUsage: 1,
			QOSClass:    v1.PodQOSBurstable,
		},
		{
			Key:         types.NamespacedName{Name: "prioirty-2"},
			Priority:    2,
			PodCPUUsage: 1,
			QOSClass:    v1.PodQOSBurstable,
		},
		{
			Key:         types.NamespacedName{Name: "prioirty-2-2"},
			Priority:    2,
			PodCPUUsage: 2,
			QOSClass:    v1.PodQOSBurstable,
		},
		{
			Key:      types.NamespacedName{Name: "priority-1"},
			Priority: 1,
			QOSClass: v1.PodQOSBurstable,
		},
		{
			Key:       types.NamespacedName{Name: "time-1"},
			StartTime: &now,
			QOSClass:  v1.PodQOSGuaranteed,
		},
		{
			Key:       types.NamespacedName{Name: "time-2"},
			StartTime: &later,
			QOSClass:  v1.PodQOSGuaranteed,
		},
	}
	CpuUsageSort(pods)
	t.Logf("sorted pods:")
	for _, p := range pods {
		t.Logf("key %s, useElasticCPU %v, qosClass %s, priority %d, usage %f, elasticCPUUsage %d, startTime %v", p.Key, (p.ElasticCPU != 0), p.QOSClass, p.Priority, p.PodCPUUsage, p.ElasticCPU, p.StartTime)
	}
}
