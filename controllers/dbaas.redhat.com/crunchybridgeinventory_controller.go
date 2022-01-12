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
	dbaasredhatcomv1alpha1 "github.com/CrunchyData/crunchy-bridge-operator/apis/dbaas.redhat.com/v1alpha1"
	"github.com/CrunchyData/crunchy-bridge-operator/internal/bridgeapi"
	"github.com/CrunchyData/crunchy-bridge-operator/internal/kubeadapter"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"net/url"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// CrunchyBridgeInventoryReconciler reconciles a CrunchyBridgeInventory object
type CrunchyBridgeInventoryReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	APIBaseURL string
	Log        logr.Logger
}

//+kubebuilder:rbac:groups=dbaas.redhat.com,resources=crunchybridgeinventories,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
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

	bridgeapiClient, err := setupClient(r.Client, inventory, r.APIBaseURL, logger)
	if err != nil {
		statusErr := r.updateStatus(ctx, inventory, metav1.ConditionFalse, AuthenticationError, err.Error())
		if statusErr != nil {
			logger.Error(statusErr, "Error in updating CrunchyBridgeInventory status")
			return ctrl.Result{Requeue: true}, statusErr
		}
		logger.Error(err, "Error while setting up CrunchyBridge Client")
		return ctrl.Result{}, err
	}
	logger.Info("Crunchy Bridge Client Configured ")
	err = r.discoverInventories(&inventory, bridgeapiClient, logger)
	if err != nil {
		statusErr := r.updateStatus(ctx, inventory, metav1.ConditionFalse, BackendError, err.Error())
		if statusErr != nil {
			logger.Error(statusErr, "Error in updating CrunchyBridgeInventory status")
			return ctrl.Result{Requeue: true}, statusErr
		}
		logger.Error(err, "Error while querying the inventory from CrunchyBridge")
		return ctrl.Result{}, err
	}
	statusErr := r.updateStatus(ctx, inventory, metav1.ConditionTrue, SyncOK, SuccessMessage)
	if statusErr != nil {
		logger.Error(statusErr, "Error in updating CrunchyBridgeInventory status")
		return ctrl.Result{Requeue: true}, statusErr
	}
	return ctrl.Result{}, nil
}

// setupClient
func setupClient(client client.Client, inventory dbaasredhatcomv1alpha1.CrunchyBridgeInventory, APIBaseURL string, logger logr.Logger) (*bridgeapi.Client, error) {
	baseUrl, err := url.Parse(APIBaseURL)
	if err != nil {
		logger.Error(err, "Malformed URL", "URL", APIBaseURL)
		return nil, err
	}
	kubeSecretProvider := &kubeadapter.KubeSecretCredentialProvider{
		Client:      client,
		Namespace:   inventory.Spec.CredentialsRef.Namespace,
		Name:        inventory.Spec.CredentialsRef.Name,
		KeyField:    KEYFIELDNAME,
		SecretField: SECRETFIELDNAME,
	}

	return bridgeapi.NewClient(baseUrl, kubeSecretProvider, bridgeapi.SetLogger(logger))
}

// updateStatus
func (r *CrunchyBridgeInventoryReconciler) updateStatus(ctx context.Context, inventory dbaasredhatcomv1alpha1.CrunchyBridgeInventory, conidtionStatus metav1.ConditionStatus, reason, message string) error {
	setStatusCondition(&inventory, SpecSynced, conidtionStatus, reason, message)
	if err := r.Status().Update(ctx, &inventory); err != nil {
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CrunchyBridgeInventoryReconciler) SetupWithManager(mgr ctrl.Manager) error {

	log := r.Log.WithValues("during", "CrunchyBridgeInventoryReconciler SetupWithManager")

	mapFn := handler.MapFunc(func(a client.Object) []ctrl.Request {
		if instance, ok := a.(*dbaasredhatcomv1alpha1.CrunchyBridgeInstance); ok {
			log.Info("Equeuing CrunchyBridgeInventory from CrunchyBridgeInstance", "Inventory", instance.Spec.InventoryRef)
			return []ctrl.Request{
				{NamespacedName: types.NamespacedName{
					Name:      instance.Spec.InventoryRef.Name,
					Namespace: instance.Spec.InventoryRef.Namespace,
				},
				}}
		}
		return nil
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&dbaasredhatcomv1alpha1.CrunchyBridgeInventory{}).
		Watches(&source.Kind{Type: &dbaasredhatcomv1alpha1.CrunchyBridgeInstance{}}, handler.EnqueueRequestsFromMapFunc(mapFn)).
		Complete(r)
}
