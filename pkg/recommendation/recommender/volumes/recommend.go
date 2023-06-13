package volumes

import (
	"fmt"

	"github.com/gocrane/crane/pkg/recommendation/framework"
)

func (vr *VolumesRecommender) PreRecommend(ctx *framework.RecommendationContext) error {
	return nil
}

func (vr *VolumesRecommender) Recommend(ctx *framework.RecommendationContext) error {
	// Check if each volume is being used by any pods
	isOrphanVolume := true
	for _, pod := range ctx.Pods {
		for _, volumeClaim := range pod.Spec.Volumes {
			if volumeClaim.PersistentVolumeClaim == nil {
				continue
			}
			for _, pvc := range ctx.PVCs {
				if volumeClaim.PersistentVolumeClaim.ClaimName == pvc.Name {
					isOrphanVolume = false
				}
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
func (vr *VolumesRecommender) Policy(ctx *framework.RecommendationContext) error {
	return nil
}
