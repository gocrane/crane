package utils

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeClient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetPodTemplate(t *testing.T) {
	ctx := context.TODO()
	testNamespace := "test-namespace"
	testName := "test-name"
	testImage := "test-image"
	testCases := []struct {
		object     client.Object
		kind       string
		apiVersion string
	}{
		{
			object: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      testName,
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: testImage,
								},
							},
						},
					},
				},
			},
			kind:       "Deployment",
			apiVersion: "apps/v1",
		},
		{
			object: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      testName,
				},
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: testImage,
								},
							},
						},
					},
				},
			},
			kind:       "StatefulSet",
			apiVersion: "apps/v1",
		},
		{
			object: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      testName,
				},
				Spec: appsv1.DaemonSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: testImage,
								},
							},
						},
					},
				},
			},
			kind:       "DaemonSet",
			apiVersion: "apps/v1",
		},
	}

	for _, tc := range testCases {
		fakeClient := fakeClient.NewClientBuilder().WithObjects(tc.object).Build()
		podTemplate, err := GetPodTemplate(ctx, testNamespace, testName, tc.kind, tc.apiVersion, fakeClient)
		if err != nil {
			t.Errorf("get pod template error: %v", err)
		}
		if len(podTemplate.Spec.Containers) == 1 && podTemplate.Spec.Containers[0].Image != testImage {
			t.Errorf("the container image of pod template is inconsistent: expect %s, actual is %s", testImage, podTemplate.Spec.Containers[0].Image)
		}
	}
}
