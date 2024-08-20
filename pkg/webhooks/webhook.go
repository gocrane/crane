/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package webhooks

import (
	"context"

	analysisapi "github.com/gocrane/api/analysis/v1alpha1"
	autoscalingapi "github.com/gocrane/api/autoscaling/v1alpha1"
	ensuranceapi "github.com/gocrane/api/ensurance/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/gocrane/crane/pkg/ensurance/config"
	"github.com/gocrane/crane/pkg/webhooks/autoscaling"
	"github.com/gocrane/crane/pkg/webhooks/ensurance"
	"github.com/gocrane/crane/pkg/webhooks/pod"
	"github.com/gocrane/crane/pkg/webhooks/prediction"
	"github.com/gocrane/crane/pkg/webhooks/recommendation"
)

func SetupWebhookWithManager(mgr ctrl.Manager, autoscalingEnabled, nodeResourceEnabled, clusterNodePredictionEnabled, analysisEnabled, timeseriespredictEnabled, qosInitializer bool, qosConfigPath string) error {
	if timeseriespredictEnabled {
		tspValidationAdmission := prediction.ValidationAdmission{}
		err := ctrl.NewWebhookManagedBy(mgr).
			For(&predictionapi.TimeSeriesPrediction{}).
			WithValidator(&tspValidationAdmission).
			Complete()
		if err != nil {
			klog.Errorf("Failed to setup tsp webhook: %v", err)
			return err
		}
	}

	if analysisEnabled {
		recomendValidationAdmission := recommendation.ValidationAdmission{}
		err := ctrl.NewWebhookManagedBy(mgr).
			For(&analysisapi.Recommendation{}).
			WithValidator(&recomendValidationAdmission).
			Complete()
		if err != nil {
			klog.Errorf("Failed to setup recommendation webhook: %v", err)
			return err
		}

		analyticsValidationAdmission := recommendation.ValidationAdmission{}
		err = ctrl.NewWebhookManagedBy(mgr).
			For(&analysisapi.Analytics{}).
			WithValidator(&analyticsValidationAdmission).
			Complete()
		if err != nil {
			klog.Errorf("Failed to setup analytics webhook: %v", err)
			return err
		}
	}

	if nodeResourceEnabled || clusterNodePredictionEnabled {
		nodeQOSValidationAdmission := ensurance.NodeQOSValidationAdmission{}
		err := ctrl.NewWebhookManagedBy(mgr).
			For(&ensuranceapi.NodeQOS{}).
			WithValidator(&nodeQOSValidationAdmission).
			Complete()
		if err != nil {
			klog.Errorf("Failed to setup NodeQOS webhook: %v", err)
			return err
		}

		actionValidationAdmission := ensurance.ActionValidationAdmission{}
		err = ctrl.NewWebhookManagedBy(mgr).
			For(&ensuranceapi.AvoidanceAction{}).
			WithValidator(&actionValidationAdmission).
			Complete()
		if err != nil {
			klog.Errorf("Failed to setup AvoidanceAction webhook: %v", err)
			return err
		}
	}

	if autoscalingEnabled {
		autoscalingValidationAdmission := autoscaling.ValidationAdmission{}
		err := ctrl.NewWebhookManagedBy(mgr).
			For(&autoscalingapi.EffectiveHorizontalPodAutoscaler{}).
			WithValidator(&autoscalingValidationAdmission).
			Complete()
		if err != nil {
			klog.Errorf("Failed to setup autoscaling webhook: %v", err)
		}
		klog.Infof("Succeed to setup autoscaling webhook")
	}

	if qosInitializer {
		qosConfig, err := config.LoadQOSConfigFromFile(qosConfigPath)
		if err != nil {
			klog.Errorf("Failed to load qos initializer config: %v", err)
		}

		podMutatingAdmission := pod.NewMutatingAdmission(qosConfig, BuildPodQosListFunction(mgr))
		err = ctrl.NewWebhookManagedBy(mgr).
			For(&corev1.Pod{}).
			WithDefaulter(podMutatingAdmission).
			Complete()
		if err != nil {
			klog.Errorf("Failed to setup qos initializer webhook: %v", err)
		}
		klog.Infof("Succeed to setup qos initializer webhook")
	}

	return nil
}

func BuildPodQosListFunction(mgr ctrl.Manager) func() ([]*ensuranceapi.PodQOS, error) {
	return func() (qosSlice []*ensuranceapi.PodQOS, err error) {
		podQOSList := ensuranceapi.PodQOSList{}
		if err := mgr.GetCache().List(context.Background(), &podQOSList); err != nil {
			return nil, err
		}
		for _, qos := range podQOSList.Items {
			qosSlice = append(qosSlice, qos.DeepCopy())
		}
		return qosSlice, err
	}
}
