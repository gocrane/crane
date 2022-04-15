package ehpa

import (
	"fmt"
	"strings"

	autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"github.com/gocrane/crane/pkg/metrics"
)

type hpaEventHandler struct {
	enqueueHandler handler.EnqueueRequestForObject
}

func (h *hpaEventHandler) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	pod := evt.Object.(*autoscalingv2.HorizontalPodAutoscaler)
	if pod.DeletionTimestamp != nil {
		h.Delete(event.DeleteEvent{Object: evt.Object}, q)
		return
	}

	h.enqueueHandler.Create(evt, q)
}

func (h *hpaEventHandler) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	h.enqueueHandler.Delete(evt, q)
}

func (h *hpaEventHandler) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	newHpa := evt.ObjectNew.(*autoscalingv2.HorizontalPodAutoscaler)
	oldHpa := evt.ObjectOld.(*autoscalingv2.HorizontalPodAutoscaler)
	klog.V(6).Infof("hpa %s OnUpdate", klog.KObj(newHpa))
	if oldHpa.Status.DesiredReplicas != newHpa.Status.DesiredReplicas {
		for _, cond := range newHpa.Status.Conditions {
			if cond.Reason == "SucceededRescale" || cond.Reason == "SucceededOverloadRescale" {
				scaleType := "hpa"
				if strings.HasPrefix("ehpa-", newHpa.Name) {
					scaleType = "ehpa"
				}

				direction := "Down"
				if newHpa.Status.DesiredReplicas > oldHpa.Status.DesiredReplicas {
					direction = "Up"
				}

				labels := map[string]string{
					"resourceName": fmt.Sprintf("%s/%s", newHpa.Namespace, newHpa.Name),
					"type":         scaleType,
					"direction":    direction,
				}
				metrics.HPAScaleCount.With(labels).Inc()

				break
			}
		}
	}
	h.enqueueHandler.Update(evt, q)
}

func (h *hpaEventHandler) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
}
