/*
Copyright 2024.

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

package v1

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var memcachedlog = logf.Log.WithName("memcached-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *Memcached) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-example-com-example-com-v1alpha1-memcached,mutating=true,failurePolicy=fail,sideEffects=None,groups=example.com.example.com,resources=memcacheds,verbs=create;update,versions=v1alpha1,name=mmemcached.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Memcached{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Memcached) Default() {
	memcachedlog.Info("default", "name", r.Name)

	// if r.Spec.Verbose == "" {
	// 	r.Spec.Verbose = Enable
	// }
}

// +kubebuilder:webhook:path=/validate-example-com-example-com-v1alpha1-memcached,mutating=false,failurePolicy=fail,sideEffects=None,groups=example.com.example.com,resources=memcacheds,verbs=create;update,versions=v1alpha1,name=vmemcached.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Memcached{}

// =================================================================================================
// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Memcached) ValidateCreate() (admission.Warnings, error) {
	memcachedlog.Info("validate create", "name", r.Name)

	return nil, r.validateMemcached()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Memcached) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	memcachedlog.Info("validate update", "name", r.Name)

	return nil, r.validateMemcached()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Memcached) ValidateDelete() (admission.Warnings, error) {
	memcachedlog.Info("validate delete", "name", r.Name)

	return nil, nil
}

// =================================================================================================
func (r *Memcached) validateMemcached() error {
	var allErrs field.ErrorList

	if err := r.validateMemcachedMemoryLimit(); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "cache.bsod.io", Kind: "Memcached"},
		r.Name, allErrs)

}

func (r *Memcached) validateMemcachedMemoryLimit() *field.Error {
	// if *r.Spec.Resources.ResourceLimits.Memory != "" {
	// 	quantity := resource.MustParse(*r.Spec.ResourceLimits.Memory)
	// 	memLimitKb := quantity.Value()
	// 	if memLimitKb < int64(268435456) {
	// 		return field.Invalid(field.NewPath("resources").Child("limits").Child("memory"), r.Spec.ResourceLimits.Memory, "must be more or equal to 256Mi")
	// 	}
	// }
	return nil
}
