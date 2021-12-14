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

package v1alpha1

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/utils/log"
)

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	predAdmission := PredictionAdmission{}
	err := ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.TimeSeriesPrediction{}).
		WithDefaulter(&predAdmission).
		WithValidator(&predAdmission).
		Complete()
	if err != nil {
		log.Logger().V(2).Info("Failed to setup webhook", err)
	}
	return err
}

type PredictionAdmission struct {
}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (p *PredictionAdmission) Default(ctx context.Context, req runtime.Object) error {
	log.Logger().Info("default", "name", req)
	return nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (p *PredictionAdmission) ValidateCreate(ctx context.Context, req runtime.Object) error {
	log.Logger().Info("validate create", "name", req)
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (p *PredictionAdmission) ValidateUpdate(ctx context.Context, old, new runtime.Object) error {
	log.Logger().Info("validate update", "name", new)

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (p *PredictionAdmission) ValidateDelete(ctx context.Context, req runtime.Object) error {
	log.Logger().Info("validate delete", "name", req)

	return nil
}
