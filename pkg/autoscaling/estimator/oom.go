package estimator

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	recommendermodel "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/recommender/model"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"

	"github.com/gocrane/crane/pkg/oom"
)

type OOMResourceEstimator struct {
	OOMRecorder oom.Recorder
}

func (e *OOMResourceEstimator) GetResourceEstimation(evpa *autoscalingapi.EffectiveVerticalPodAutoscaler, config map[string]string, containerName string, currRes *corev1.ResourceRequirements) (corev1.ResourceList, error) {
	oomRecords, err := e.OOMRecorder.GetOOMRecord()
	if err != nil {
		return nil, err
	}

	var oomRecord *oom.OOMRecord
	podPrefix := fmt.Sprintf("%s-", evpa.Spec.TargetRef.Name)
	for _, record := range oomRecords {
		if strings.HasPrefix(record.Pod, podPrefix) && containerName == record.Container {
			oomRecord = &record
		}
	}

	// ignore too old oom events
	if oomRecord != nil && time.Now().Sub(oomRecord.OOMAt) <= (time.Hour*24*7) {
		memoryOOM := oomRecord.Memory.Value()
		recommendResource := corev1.ResourceList{}
		var memoryNeeded recommendermodel.ResourceAmount

		bumpUpRatio := config[fmt.Sprintf("workload.%s", evpa.Spec.TargetRef.Name)]
		if bumpUpRatio == "" {
			memoryNeeded = recommendermodel.ResourceAmountMax(recommendermodel.ResourceAmount(memoryOOM)+recommendermodel.MemoryAmountFromBytes(recommendermodel.OOMMinBumpUp),
				recommendermodel.ScaleResource(recommendermodel.ResourceAmount(memoryOOM), recommendermodel.OOMBumpUpRatio))

		} else {
			oomBumpUpRatio, err := strconv.ParseFloat(bumpUpRatio, 64)
			if err != nil {
				return nil, fmt.Errorf("Parse bumpUpRatio failed: %v. ", err)
			}
			memoryNeeded = recommendermodel.ScaleResource(recommendermodel.ResourceAmount(memoryOOM), oomBumpUpRatio)
		}

		recommendResource[corev1.ResourceMemory] = *resource.NewQuantity(int64(memoryNeeded), resource.BinarySI)
		return recommendResource, nil
	}

	return nil, nil
}

func (e *OOMResourceEstimator) DeleteEstimation(evpa *autoscalingapi.EffectiveVerticalPodAutoscaler) {
	// do nothing
	return
}
