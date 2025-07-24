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

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kvctlv1 "github.com/CaptainIRS/sharded-kvs/api/v1"
)

// KVStoreReconciler reconciles a KVStore object
type KVStoreReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kvctl.captainirs.dev,resources=kvstores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kvctl.captainirs.dev,resources=kvstores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kvctl.captainirs.dev,resources=kvstores/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KVStore object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *KVStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var kvStore kvctlv1.KVStore
	if err := r.Get(ctx, req.NamespacedName, &kvStore); err != nil {
		log.Error(err, "unable to fetch KVStore")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var kvStoreList kvctlv1.KVStoreList
	if err := r.List(ctx, &kvStoreList); err != nil {
		log.Error(err, "unable to fetch KVStore list")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KVStoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kvctlv1.KVStore{}).
		Named("kvstore").
		Complete(r)
}
