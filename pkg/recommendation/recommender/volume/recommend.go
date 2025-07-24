package volume

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/gocrane/crane/pkg/recommendation/framework"
)

func (vr *VolumeRecommender) PreRecommend(ctx *framework.RecommendationContext) error {
	return nil
}

func (vr *VolumeRecommender) Recommend(ctx *framework.RecommendationContext) error {
	// Check if each volume is being used by any pods
	isOrphanVolume := true
	var pv corev1.PersistentVolume
	if err := framework.ObjectConversion(ctx.Object, &pv); err != nil {
		return err
	}
	for _, pod := range ctx.Pods {
		for _, volumeClaim := range pod.Spec.Volumes {
			if volumeClaim.PersistentVolumeClaim == nil {
				continue
			}
			if volumeClaim.PersistentVolumeClaim.ClaimName == pv.Spec.ClaimRef.Name {
				isOrphanVolume = false
			}
		}
	}
	if !isOrphanVolume {
		return fmt.Errorf("Volume %s is not an orphan volume ", ctx.Object.GetName())
	}
	ctx.Recommendation.Status.Action = "Delete"
	ctx.Recommendation.Status.Description = "It is an Orphan Volumes"
	return nil
}

// Policy add some logic for result of recommend phase.
func (vr *VolumeRecommender) Policy(ctx *framework.RecommendationContext) error {
	return nil
}
