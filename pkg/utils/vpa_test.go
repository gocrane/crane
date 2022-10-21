package utils

import (
	"context"
	"errors"
	"reflect"
	"testing"

	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type mockClient struct {
	client.Client
}

type mockErrorClient struct {
	client.Client
}

func (c mockErrorClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return errors.New("raise an error")
}

func (c mockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return nil
}

func TestGetEVPAFromScaleTarget(t *testing.T) {
	tests := []struct {
		name       string
		context    context.Context
		kubeClient client.Client
		namespace  string
		objRef     corev1.ObjectReference
		want       *autoscalingapi.EffectiveVerticalPodAutoscaler
		wantErr    bool
	}{
		{
			name:       "base",
			context:    context.Background(),
			kubeClient: mockErrorClient{},
			namespace:  "default",
			wantErr:    true,
		},
		{
			name:       "list without error",
			context:    context.Background(),
			kubeClient: mockClient{},
			namespace:  "default",
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetEVPAFromScaleTarget(tt.context, tt.kubeClient, tt.namespace, tt.objRef)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetEVPAFromScaleTarget() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetEVPAFromScaleTarget() = %v, want %v", got, tt.want)
			}
		})
	}
}
