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
	"github.com/CrunchyData/crunchy-bridge-operator/controllers/resources"
	dbaasv1alpha1 "github.com/RHEcosystemAppEng/dbaas-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

	err = r.updateStatus(&inventory, connectionAPIKeys)
	if err != nil {
		logger.Error(err, "error fetching instances status")
		return ctrl.Result{}, err
	}
	UpdateCondition(&inventory, "crunchybridgeinventory", metav1.ConditionTrue, "Created")
	if err = r.Status().Update(ctx, &inventory); err != nil {
		if apierrors.IsConflict(err) {
			logger.Info("conflict at status update, requeue to sort it out")
			return ctrl.Result{}, nil
		} else {
			logger.Error(err, "error saving modified CrunchyBridgeInventory status")

			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CrunchyBridgeInventoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dbaasredhatcomv1alpha1.CrunchyBridgeInventory{}).
		Complete(r)
}

func (r *CrunchyBridgeInventoryReconciler) updateStatus(dbaasredhatcomv1alpha1 *dbaasredhatcomv1alpha1.CrunchyBridgeInventory, keys resources.ConnectionAPIKeys) error {
	var bridgeInstances []dbaasv1alpha1.Instance
	instance := dbaasv1alpha1.Instance{
		InstanceID:   "azpiatrcn5eujmhncap73ujeqm",
		Name:         "example-cluster",
		InstanceInfo: map[string]string{"provider_id": "aws", "region_id": "us-east-1", "type": "primary"},
	}

	bridgeInstances = append(bridgeInstances, instance)
	dbaasredhatcomv1alpha1.Status.Instances = bridgeInstances
	dbaasredhatcomv1alpha1.Status.Type = "CrunchyBridge Postgres"
	return nil
}

// UpdateCondition will update or add the provided condition.
func UpdateCondition(cr *dbaasredhatcomv1alpha1.CrunchyBridgeInventory, conditionType string, status metav1.ConditionStatus, reason string) {

	message := fmt.Sprintf("%s inventory listed successfully", cr.Name)
	found := false

	condition := &metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}
	conditions := []metav1.Condition{}

	for _, c := range cr.Status.Conditions {
		if condition.Type != c.Type {
			conditions = append(conditions, c)
			continue
		}
		if c.Status != condition.Status {
			c.Status = condition.Status
			c.LastTransitionTime = condition.LastTransitionTime
		}
		if c.Reason != condition.Reason {
			c.Reason = condition.Reason
		}
		if c.Message != condition.Message {
			c.Message = condition.Message
		}
		conditions = append(conditions, c)
		found = true
	}
	cr.Status.Conditions = conditions
	if !found {
		conditions = append(conditions, *condition)
	}
	cr.Status.Conditions = conditions

}
