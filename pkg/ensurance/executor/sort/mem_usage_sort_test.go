package sort

import (
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/gocrane/crane/pkg/ensurance/executor/podinfo"
)

func TestMemUsageSorter(t *testing.T) {
	now := metav1.NewTime(time.Unix(1000, 0).UTC())
	later := metav1.NewTime(time.Unix(2000, 0).UTC())

	pods := []podinfo.PodContext{
		{
			Key:             types.NamespacedName{Name: "elastic-mem-2"},
			ElasticMemLimit: 2,
			QOSClass:        v1.PodQOSBestEffort,
		},
		{
			Key:             types.NamespacedName{Name: "elastic-mem-4"},
			ElasticMemLimit: 4,
			QOSClass:        v1.PodQOSBestEffort,
		},
		{
			Key:         types.NamespacedName{Name: "mem-1"},
			PodMemUsage: 1,
			QOSClass:    v1.PodQOSGuaranteed,
		},
		{
			Key:         types.NamespacedName{Name: "mem-2"},
			PodMemUsage: 2,
			QOSClass:    v1.PodQOSBurstable,
		},
		{
			Key:         types.NamespacedName{Name: "guarantee-1"},
			PodMemUsage: 1,
			QOSClass:    v1.PodQOSGuaranteed,
		},
		{
			Key:         types.NamespacedName{Name: "burstable-2"},
			PodMemUsage: 1,
			QOSClass:    v1.PodQOSBurstable,
		},
		{
			Key:         types.NamespacedName{Name: "prioirty-2"},
			Priority:    2,
			PodMemUsage: 1,
			QOSClass:    v1.PodQOSBurstable,
		},
		{
			Key:         types.NamespacedName{Name: "prioirty-2-2"},
			Priority:    2,
			PodMemUsage: 2,
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
	MemUsageSort(pods)
	t.Logf("sorted pods:")
	for _, p := range pods {
		t.Logf("key %s, useElasticMem %v, qosClass %s, priority %d, usage %f, elasticMemUsage %d, startTime %v", p.Key, (p.ElasticMemLimit != 0), p.QOSClass, p.Priority, p.PodMemUsage, p.ElasticMemLimit, p.StartTime)
	}
}
