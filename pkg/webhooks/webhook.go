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
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"

	analysisapi "github.com/gocrane/api/analysis/v1alpha1"
	predictionapi "github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/webhooks/prediction"
	"github.com/gocrane/crane/pkg/webhooks/recommendation"
)

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	tspValidationAdmission := prediction.ValidationAdmission{}
	err := ctrl.NewWebhookManagedBy(mgr).
		For(&predictionapi.TimeSeriesPrediction{}).
		WithValidator(&tspValidationAdmission).
		Complete()
	if err != nil {
		klog.Errorf("Failed to setup tsp webhook: %v", err)
	}

	recomendValidationAdmission := recommendation.ValidationAdmission{}
	err = ctrl.NewWebhookManagedBy(mgr).
		For(&analysisapi.Recommendation{}).
		WithValidator(&recomendValidationAdmission).
		Complete()
	if err != nil {
		klog.Errorf("Failed to setup recommendation webhook: %v", err)
	}

	analyticsValidationAdmission := recommendation.ValidationAdmission{}
	err = ctrl.NewWebhookManagedBy(mgr).
		For(&analysisapi.Analytics{}).
		WithValidator(&analyticsValidationAdmission).
		Complete()
	if err != nil {
		klog.Errorf("Failed to setup analytics webhook: %v", err)
	}

	return err
}
