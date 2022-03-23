package oom

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gocrane/crane/pkg/known"
	"github.com/gocrane/crane/pkg/metrics"
)

const (
	ConfigMapOOMRecordName = "oom-record"
	ConfigMapDataOOMRecord = "data"
)

type Recorder interface {
	// GetOOMRecord get OOMRecord list that stored in configmap
	GetOOMRecord() ([]OOMRecord, error)
}

type OOMRecord struct {
	Pod       string
	Container string
	Memory    resource.Quantity
	OOMAt     time.Time
}

// PodOOMRecorder is responsible for record pod oom event in configmap
type PodOOMRecorder struct {
	mu sync.Mutex

	client.Client
	queue workqueue.Interface
	cache []OOMRecord
}

func (r *PodOOMRecorder) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.V(10).Infof("Got pod %s", req.NamespacedName)

	pod := &v1.Pod{}
	err := r.Client.Get(ctx, req.NamespacedName, pod)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if IsOOMKilled(pod) {
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.LastTerminationState.Terminated != nil && cs.LastTerminationState.Terminated.Reason == "OOMKilled" {
				for _, container := range pod.Spec.Containers {
					if cs.Name == container.Name {
						// don't handle if request is not set
						if mem, ok := container.Resources.Requests[v1.ResourceMemory]; ok {
							labels := map[string]string{
								"pod":       pod.Name,
								"container": cs.Name,
							}
							metrics.OOMCount.With(labels).Inc()
							r.queue.Add(OOMRecord{
								Pod:       pod.Name,
								Container: cs.Name,
								Memory:    mem,
								OOMAt:     cs.LastTerminationState.Terminated.FinishedAt.Time,
							})
							klog.V(2).Infof("pod name %s, container name %s, memory %v,oom happens!", pod.Name, cs.Name, mem)
						}
					}
				}
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *PodOOMRecorder) GetOOMRecord() ([]OOMRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cache == nil {
		oomConfigMap := &v1.ConfigMap{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{Namespace: known.CraneSystemNamespace, Name: ConfigMapOOMRecordName}, oomConfigMap)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		var oomRecords []OOMRecord
		return oomRecords, json.Unmarshal([]byte(oomConfigMap.Data[ConfigMapDataOOMRecord]), &oomRecords)
	}

	return r.cache, nil
}

func (r *PodOOMRecorder) Run(stopCh <-chan struct{}) error {
	for {
		select {
		case <-stopCh:
			return nil
		default:
		}

		o, quit := r.queue.Get()
		if quit {
			return errors.New("queue of OOMRecord recorder is shutting down, this should not happen")
		}

		oomRecords, err := r.GetOOMRecord()
		if err != nil {
			klog.Errorf("Get oomRecord failed: %v", err)
			r.queue.Add(o)
		}
		err = r.updateOOMRecord(o.(OOMRecord), oomRecords)
		if err != nil {
			klog.Errorf("Update oomRecord failed: %v", err)
			r.queue.Add(o)
		} else {
			r.queue.Done(o)
		}
	}
}

func (r *PodOOMRecorder) updateOOMRecord(oomRecord OOMRecord, saved []OOMRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	isFound := false
	isUpdated := false

	if saved == nil {
		// handle nil
		saved = []OOMRecord{}
	}
	for index := range saved {
		if saved[index].Pod == oomRecord.Pod && saved[index].Container == oomRecord.Container {
			isFound = true
			if oomRecord.Memory.Value() > saved[index].Memory.Value() {
				saved[index].Memory = oomRecord.Memory
				saved[index].OOMAt = oomRecord.OOMAt
			}
		}
	}

	if !isFound {
		saved = append(saved, oomRecord)
		isUpdated = true
	}

	if isUpdated {
		r.cache = saved
		savedData, err := json.Marshal(saved)
		if err != nil {
			return err
		}

		configMap := &v1.ConfigMap{}
		err = r.Client.Get(context.TODO(), types.NamespacedName{Namespace: known.CraneSystemNamespace, Name: ConfigMapOOMRecordName}, configMap)
		if err != nil {
			if apierrors.IsNotFound(err) {
				// create ConfigMap if not exist
				configMap.Name = ConfigMapOOMRecordName
				configMap.Namespace = known.CraneSystemNamespace
				configMap.Data = make(map[string]string)
				configMap.Data[ConfigMapDataOOMRecord] = string(savedData)
				return r.Client.Create(context.TODO(), configMap)
			}

			configMap.Data = make(map[string]string)
			configMap.Data[ConfigMapDataOOMRecord] = string(savedData)
			return err
		}

		configMap.Data = make(map[string]string)
		configMap.Data[ConfigMapDataOOMRecord] = string(savedData)
		return r.Client.Update(context.TODO(), configMap)
	}
	return nil
}

func (r *PodOOMRecorder) SetupWithManager(mgr ctrl.Manager) error {
	r.queue = workqueue.New()
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Pod{}).
		Complete(r)
}

func IsOOMKilled(pod *v1.Pod) bool {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.RestartCount > 0 &&
			containerStatus.LastTerminationState.Terminated != nil &&
			containerStatus.LastTerminationState.Terminated.Reason == "OOMKilled" {
			return true
		}
	}

	return false
}
