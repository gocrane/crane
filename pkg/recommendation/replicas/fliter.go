package replicas

import (
	"github.com/gocrane/crane/pkg/recommendation/framework"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func (rr *ReplicasRecommender) Filter(ctx *framework.RecommendationContext) error {
	// 1. convert kubernetes object to gvk object
	obj := ctx.Object
	var (
		u   unstructured.Unstructured
		err error
	)
	u.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return err
	}
	gvk := u.GroupVersionKind()
	apiv, k := gvk.ToAPIVersionAndKind()
	// 2. load recommender accepted kubernetes object

	// 3. if not support, abort the recommendation flow

	return nil
}
