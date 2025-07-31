/*
Copyright 2025.

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
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kvctlv1 "github.com/CaptainIRS/sharded-kvs/api/v1"
)

// nolint:unused
// log is for logging in this package.
var kvstorelog = logf.Log.WithName("kvstore-resource")

// SetupKVStoreWebhookWithManager registers the webhook for KVStore in the manager.
func SetupKVStoreWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&kvctlv1.KVStore{}).
		WithValidator(&KVStoreCustomValidator{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-kvctl-captainirs-dev-v1-kvstore,mutating=false,failurePolicy=fail,sideEffects=None,groups=kvctl.captainirs.dev,resources=kvstores,verbs=create;update,versions=v1,name=vkvstore-v1.kb.io,admissionReviewVersions=v1

// KVStoreCustomValidator struct is responsible for validating the KVStore resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type KVStoreCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &KVStoreCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type KVStore.
func (v *KVStoreCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	kvstore, ok := obj.(*kvctlv1.KVStore)
	if !ok {
		return nil, fmt.Errorf("expected a KVStore object but got %T", obj)
	}
	kvstorelog.Info("Validation for KVStore upon creation", "name", kvstore.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type KVStore.
func (v *KVStoreCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	kvstore, ok := newObj.(*kvctlv1.KVStore)
	if !ok {
		return nil, fmt.Errorf("expected a KVStore object for the newObj but got %T", newObj)
	}
	kvstorelog.Info("Validation for KVStore upon update", "name", kvstore.GetName())

	oldKvstore, ok := oldObj.(*kvctlv1.KVStore)
	if !ok {
		return nil, fmt.Errorf("expected a KVStore object for the oldObj but got %T", newObj)
	}
	kvstorelog.Info("Validation for KVStore upon update", "name", kvstore.GetName())

	if oldKvstore.Status.Phase != kvctlv1.PhaseNormal {
		return nil, fmt.Errorf("cannot change while shards are redistributing")
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type KVStore.
func (v *KVStoreCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	kvstore, ok := obj.(*kvctlv1.KVStore)
	if !ok {
		return nil, fmt.Errorf("expected a KVStore object but got %T", obj)
	}
	kvstorelog.Info("Validation for KVStore upon deletion", "name", kvstore.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
