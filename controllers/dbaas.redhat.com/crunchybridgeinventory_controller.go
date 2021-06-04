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

package dbaasredhatcom

import (
	"context"
	"github.com/CrunchyData/crunchy-bridge-operator/controllers/resources"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	dbaasredhatcomv1alpha1 "github.com/CrunchyData/crunchy-bridge-operator/apis/dbaas.redhat.com/v1alpha1"
)

// CrunchyBridgeInventoryReconciler reconciles a CrunchyBridgeInventory object
type CrunchyBridgeInventoryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=dbaas.redhat.com,resources=crunchybridgeinventories,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dbaas.redhat.com,resources=crunchybridgeinventories/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dbaas.redhat.com,resources=crunchybridgeinventories/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CrunchyBridgeInventory object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *CrunchyBridgeInventoryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "CrunchyBridgeInventory", req.NamespacedName)

	var inventory dbaasredhatcomv1alpha1.CrunchyBridgeInventory
	err := r.Get(ctx, req.NamespacedName, &inventory)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// CR deleted since request queued, child objects getting GC'd, no requeue
			logger.Info("CrunchyBridgeInventory resource not found, has been deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Error fetching CrunchyBridgeInventory for reconcile")
		return ctrl.Result{}, err
	}

	// read the API Key from secret
	connectionAPIKeys, err := resources.ReadAPIKeysFromSecret(r.Client, ctx, &inventory)
	if err != nil {
		// error fetching connectionAPIKeys
		logger.Error(err, "error fetching matching Secret")
		return ctrl.Result{}, err
	}
	//TODO need to remove from logs once utiliaze the connectionAPIKeys
	logger.Info("CrunchyBridgeInventory ", "connectionAPIKeys", connectionAPIKeys)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CrunchyBridgeInventoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dbaasredhatcomv1alpha1.CrunchyBridgeInventory{}).
		Complete(r)
}
