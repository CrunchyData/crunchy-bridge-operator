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
	"fmt"

	dbaasv1alpha1 "github.com/RHEcosystemAppEng/dbaas-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	dbaasredhatcomv1alpha1 "github.com/CrunchyData/crunchy-bridge-operator/apis/dbaas.redhat.com/v1alpha1"
)

// CrunchyBridgeConnectionReconciler reconciles a CrunchyBridgeConnection object
type CrunchyBridgeConnectionReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Clientset  *kubernetes.Clientset
	APIBaseURL string
}

//+kubebuilder:rbac:groups=dbaas.redhat.com,resources=crunchybridgeconnections,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dbaas.redhat.com,resources=crunchybridgeconnections/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dbaas.redhat.com,resources=crunchybridgeconnections/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CrunchyBridgeConnection object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *CrunchyBridgeConnectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "CrunchyBridgeConnection", req.NamespacedName)

	var connection dbaasredhatcomv1alpha1.CrunchyBridgeConnection
	err := r.Get(ctx, req.NamespacedName, &connection)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// CR deleted since request queued, child objects getting GC'd, no requeue
			logger.Info("CrunchyBridgeConnection resource not found, has been deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Error fetching CrunchyBridgeConnection for reconcile")
		return ctrl.Result{}, err
	}
	inventory := dbaasredhatcomv1alpha1.CrunchyBridgeInventory{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: connection.Spec.InventoryRef.Namespace, Name: connection.Spec.InventoryRef.Name}, &inventory); err != nil {
		if apierrors.IsNotFound(err) {
			statusErr := r.updateStatus(ctx,connection, metav1.ConditionFalse, InventoryNotFound, err.Error())
			if statusErr != nil {
				logger.Error(statusErr, "Error in updating CrunchyBridgeConnection status")
				return ctrl.Result{Requeue: true}, statusErr
			}
			logger.Info("inventory resource not found, has been deleted")
			return ctrl.Result{}, err
		}
		logger.Error(err, "Error fetching CrunchyBridgeConnection for reconcile")
		return ctrl.Result{}, err
	}

	instance, err := getInstance(&inventory, connection.Spec.InstanceID)
	if instance == nil {
		statusErr := r.updateStatus(ctx, connection, metav1.ConditionFalse, NotFound, err.Error())
		if statusErr != nil {
			logger.Error(statusErr, "Error in updating CrunchyBridgeConnection status")
			return ctrl.Result{Requeue: true}, statusErr
		}
		return ctrl.Result{}, err
	}
	bridgeapiClient, err := setupClient(r.Client, inventory, r.APIBaseURL, logger)
	if err != nil {
		statusErr := r.updateStatus(ctx, connection, metav1.ConditionFalse, BackendError, err.Error())
		if statusErr != nil {
			logger.Error(statusErr, "Error in updating CrunchyBridgeConnection status")
			return ctrl.Result{Requeue: true}, statusErr
		}
		logger.Error(err, "Error while setting up CrunchyBridge Client")
		return ctrl.Result{}, err
	}

	logger.Info("Crunchy Bridge Client Configured ")
	err = r.connectionDetails(instance.InstanceID, &connection, bridgeapiClient, req, logger)
	if err != nil {
		statusErr := r.updateStatus(ctx, connection, metav1.ConditionFalse, BackendError, err.Error())
		if statusErr != nil {
			logger.Error(statusErr, "Error in updating CrunchyBridgeConnection status")
			return ctrl.Result{Requeue: true}, statusErr
		}
		logger.Error(err, "Error while getting connection details")
		return ctrl.Result{}, err
	}
	statusErr := r.updateStatus(ctx, connection, metav1.ConditionTrue, Ready, SuccessConnection)
	if statusErr != nil {
		logger.Error(statusErr, "Error in updating CrunchyBridgeInventory status")
		return ctrl.Result{Requeue: true}, statusErr
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CrunchyBridgeConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dbaasredhatcomv1alpha1.CrunchyBridgeConnection{}).
		Complete(r)
}

// getInstance returns an instance from the inventory based on instanceID
func getInstance(inventory *dbaasredhatcomv1alpha1.CrunchyBridgeInventory, instanceID string) (*dbaasv1alpha1.Instance, error) {
	if !isInventoryReady(inventory) {
		return nil, fmt.Errorf("CrunchyBridgeInventory CR status is not ready ")
	}
	for _, instance := range inventory.Status.Instances {
		if instance.InstanceID == instanceID {
			//Found the instance based on its ID
			return &instance, nil
		}
	}
	return nil, fmt.Errorf("instance id %q not found in CrunchyBridgeInventory status", instanceID)
}

// updateStatus
func (r *CrunchyBridgeConnectionReconciler) updateStatus(ctx context.Context, connection dbaasredhatcomv1alpha1.CrunchyBridgeConnection, conidtionStatus metav1.ConditionStatus, reason, message string) error {
	setStatusCondition(&connection, ReadyForBinding, conidtionStatus, reason, message)
	if err := r.Client.Status().Update(context.Background(), &connection); err != nil {
		return err
	}
	return nil
}

// isInventoryReady is the CrunchyBridgeInventory ready?
func isInventoryReady(inventory *dbaasredhatcomv1alpha1.CrunchyBridgeInventory) bool {
	cond := GetInventoryCondition(inventory, string(SpecSynced))
	return cond != nil && cond.Status == metav1.ConditionTrue
}
