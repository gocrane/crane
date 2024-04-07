package recommendation

import (
	analysisv1alph1 "github.com/gocrane/api/analysis/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"
)

func TestRecommendationIndex_GetRecommendation(t *testing.T) {
	type fields struct {
		recommendationList analysisv1alph1.RecommendationList
	}
	type args struct {
		id ObjectIdentity
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   *analysisv1alph1.Recommendation
	}{
		{
			name: "TestRecommendationIndex_GetRecommendation good case",
			fields: fields{
				recommendationList: analysisv1alph1.RecommendationList{
					Items: []analysisv1alph1.Recommendation{
						{
							ObjectMeta: v1.ObjectMeta{
								Name:      "test-recommendation-rule",
								Namespace: "test-namespace",
							},
							Spec: analysisv1alph1.RecommendationSpec{
								TargetRef: corev1.ObjectReference{
									Namespace:  "test-namespace",
									Kind:       "Deployment",
									Name:       "test-deployment-bar",
									APIVersion: "app/v1",
								},
								Type: analysisv1alph1.AnalysisTypeResource,
							},
						},
						{
							ObjectMeta: v1.ObjectMeta{
								Name:      "test-recommendation-rule",
								Namespace: "test-namespace",
							},
							Spec: analysisv1alph1.RecommendationSpec{
								TargetRef: corev1.ObjectReference{
									Namespace:  "test-namespace",
									Kind:       "Deployment",
									Name:       "test-deployment-foo",
									APIVersion: "app/v1",
								},
								Type: analysisv1alph1.AnalysisTypeResource,
							},
						},
					},
				},
			},
			want: &analysisv1alph1.Recommendation{
				ObjectMeta: v1.ObjectMeta{
					Name:      "test-recommendation-rule",
					Namespace: "test-namespace",
				},
				Spec: analysisv1alph1.RecommendationSpec{
					TargetRef: corev1.ObjectReference{
						Namespace:  "test-namespace",
						Kind:       "Deployment",
						Name:       "test-deployment-name",
						APIVersion: "app/v1",
					},
				},
			},
			args: args{
				id: ObjectIdentity{
					Name:        "test-deployment-name",
					Namespace:   "test-namespace",
					APIVersion:  "app/v1",
					Kind:        "Deployment",
					Recommender: "Resource",
				},
			},
		},
		{
			name: "TestRecommendationIndex_GetRecommendation empty case",
			fields: fields{
				recommendationList: analysisv1alph1.RecommendationList{
					Items: []analysisv1alph1.Recommendation{},
				},
			},
			args: args{
				id: ObjectIdentity{
					Name:        "test-deployment-name",
					Namespace:   "test-namespace",
					APIVersion:  "app/v1",
					Kind:        "Deployment",
					Recommender: "Resources",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := NewRecommendationIndex(tt.fields.recommendationList)
			if got := idx.GetRecommendation(tt.args.id); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetRecommendation() = %v, want %v", got, tt.want)
			}
		})
	}
}
