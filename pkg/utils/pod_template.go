package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetPodTemplate(context context.Context, namespace string, name string, kind string, apiVersion string, kubeClient client.Client) (*v1.PodTemplateSpec, error) {
	templateSpec := &v1.PodTemplateSpec{}

	key := client.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}

	if kind == "Deployment" && strings.HasPrefix(apiVersion, "apps") {
		deployment := &appsv1.Deployment{}
		err := kubeClient.Get(context, key, deployment)
		if err != nil {
			return nil, err
		}
		templateSpec = &deployment.Spec.Template
	} else if kind == "StatefulSet" && strings.HasPrefix(apiVersion, "apps") {
		statefulSet := &appsv1.StatefulSet{}
		err := kubeClient.Get(context, key, statefulSet)
		if err != nil {
			return nil, err
		}
		templateSpec = &statefulSet.Spec.Template
	} else {
		unstructed := &unstructured.Unstructured{}
		unstructed.SetAPIVersion(apiVersion)
		unstructed.SetKind(kind)
		err := kubeClient.Get(context, key, unstructed)
		if err != nil {
			return nil, err
		}

		template, found, err := unstructured.NestedMap(unstructed.Object, "spec", "template")
		if !found || err != nil {
			return nil, fmt.Errorf("get template from unstructed object %s failed. ", klog.KObj(unstructed))
		}

		templateBytes, err := json.Marshal(template)
		if err != nil {
			return nil, fmt.Errorf("marshal unstructed object failed: %v. ", err)
		}

		err = json.Unmarshal(templateBytes, templateSpec)
		if err != nil {
			return nil, fmt.Errorf("unmarshal template bytes failed: %v. ", err)
		}
	}

	return templateSpec, nil
}
