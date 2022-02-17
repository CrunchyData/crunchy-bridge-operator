/*
Copyright 2021 Crunchy Data Solutions, Inc.

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
	"errors"
	"fmt"
	"strconv"
	"time"

	dbaasv1alpha1 "github.com/RHEcosystemAppEng/dbaas-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	crunchybridgev1alpha1 "github.com/CrunchyData/crunchy-bridge-operator/apis/crunchybridge/v1alpha1"
	dbaasredhatcomv1alpha1 "github.com/CrunchyData/crunchy-bridge-operator/apis/dbaas.redhat.com/v1alpha1"
	"github.com/CrunchyData/crunchy-bridge-operator/internal/bridgeapi"
)

const (
	instanceFinalizer = "dbaas.redhat.com/crunchybridgeinstance-finalizer"
	WatchInt          = 10 * time.Second
)

// CrunchyBridgeInstanceReconciler reconciles a CrunchyBridgeInstance object
type CrunchyBridgeInstanceReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	APIBaseURL string
}

//+kubebuilder:rbac:groups=dbaas.redhat.com,resources=crunchybridgeinstances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dbaas.redhat.com,resources=crunchybridgeinstances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dbaas.redhat.com,resources=crunchybridgeinstances/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the CrunchyBridgeInstance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *CrunchyBridgeInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	instanceObj := &dbaasredhatcomv1alpha1.CrunchyBridgeInstance{}
	if err := r.Get(ctx, req.NamespacedName, instanceObj); err != nil {
		if apierrors.IsNotFound(err) {
			// Likely deleted before action or extra pass post-deletion, no-op
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Error fetching CrunchyBridgeInstance object for reconciliation")
		return ctrl.Result{}, err
	}
	inventory := dbaasredhatcomv1alpha1.CrunchyBridgeInventory{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: instanceObj.Spec.InventoryRef.Namespace, Name: instanceObj.Spec.InventoryRef.Name}, &inventory); err != nil {
		if apierrors.IsNotFound(err) {
			statusErr := r.updateStatus(instanceObj, metav1.ConditionFalse, InventoryNotFound, err.Error())
			if statusErr != nil {
				logger.Error(statusErr, "Error in updating CrunchyBridgeInstance status")
				return ctrl.Result{Requeue: true}, statusErr
			}
			logger.Info("inventory resource not found, has been deleted")
			return ctrl.Result{}, err
		}
		logger.Error(err, "Error fetching CrunchyBridgeInstance object for reconciliation")
		return ctrl.Result{}, err
	}
	bridgeapiClient, err := setupClient(r.Client, inventory, r.APIBaseURL, logger)
	if err != nil {
		statusErr := r.updateStatus(instanceObj, metav1.ConditionFalse, AuthenticationError, err.Error())
		if statusErr != nil {
			logger.Error(statusErr, "Error in updating CrunchyBridgeInstance status")
			return ctrl.Result{Requeue: true}, statusErr
		}
		logger.Error(err, "No CrunchyBridge client configured")
		return ctrl.Result{}, err
	}
	logger.Info("Crunchy Bridge Client Configured ")

	if instanceObj.DeletionTimestamp != nil && !instanceObj.DeletionTimestamp.IsZero() {
		// Cluster deletion request / process finalizer
		if listContains(instanceObj.Finalizers, instanceFinalizer) {
			if id := instanceObj.Status.InstanceID; id != "" {
				logger.Info("deleting cluster", "id", id)
				instanceObj.Status.Phase = crunchybridgev1alpha1.PhaseDeleting
				if err := r.Status().Update(ctx, instanceObj); err != nil {
					if apierrors.IsConflict(err) {
						logger.Info("Instance modified, retry reconciling")
						return ctrl.Result{Requeue: true}, nil
					}
					logger.Error(err, "Failed to update Instance phase in status")
					return ctrl.Result{}, err
				}
				if err := bridgeapiClient.DeleteCluster(id); err != nil {
					logger.Error(err, "Failed to delete a cluster")

					return ctrl.Result{}, err
				}
				logger.Info("cluster deleted", "id", id)
			}

			controllerutil.RemoveFinalizer(instanceObj, instanceFinalizer)
			if err := r.Update(ctx, instanceObj); err != nil {
				return ctrl.Result{}, err
			}

		}

	} else {
		switch instanceObj.Status.Phase {
		case crunchybridgev1alpha1.PhaseUnknown:
			// New object, add our finalizer
			if !listContains(instanceObj.Finalizers, instanceFinalizer) {
				controllerutil.AddFinalizer(instanceObj, instanceFinalizer)
				if err := r.Update(ctx, instanceObj); err != nil {
					logger.Error(err, "Failed to add finalizer to Instance")
					return ctrl.Result{}, err
				}
			}

			// Set pending phase after so any errors in setting finalizer
			// don't advance state
			instanceObj.Status.Phase = crunchybridgev1alpha1.PhasePending
			if err := r.Status().Update(ctx, instanceObj); err != nil {
				return ctrl.Result{}, err
			}

		case crunchybridgev1alpha1.PhasePending:
			req, err := r.createFromSpec(instanceObj.Spec, bridgeapiClient)
			if err != nil {
				return ctrl.Result{}, err
			}

			logger.Info("cluster creation request", "request", req)

			if err := bridgeapiClient.CreateCluster(req); err != nil {
				statusErr := r.updateStatus(instanceObj, metav1.ConditionFalse, BackendError, err.Error())
				if statusErr != nil {
					logger.Error(statusErr, "Error in updating CrunchyBridgeInstance status")
					return ctrl.Result{Requeue: true}, statusErr
				}
				return ctrl.Result{}, err
			}

			// Assuming the request was sent, update phase
			instanceObj.Status.Phase = crunchybridgev1alpha1.PhaseCreating
			if err := r.Status().Update(ctx, instanceObj); err != nil {
				return ctrl.Result{}, err
			}

		case crunchybridgev1alpha1.PhaseCreating:
			var detC bridgeapi.ClusterDetail
			if cid := instanceObj.Status.InstanceID; cid == "" {
				c, err := bridgeapiClient.ClusterByName(instanceObj.Spec.Name)
				if err != nil {
					return ctrl.Result{}, err
				}
				detC = c
			} else {
				c, err := bridgeapiClient.ClusterDetail(cid)
				if err != nil {
					return ctrl.Result{}, err
				}
				detC = c
			}
			logger.Info("cluster creating", "name", instanceObj.Spec.Name)

			if err := r.updateStatusFromDetail(detC, &instanceObj.Status); err != nil {
				statusErr := r.updateStatus(instanceObj, metav1.ConditionFalse, BackendError, err.Error())
				if statusErr != nil {
					logger.Error(statusErr, "Error in updating CrunchyBridgeInstance status")
					return ctrl.Result{Requeue: true}, statusErr
				}
				return ctrl.Result{}, err
			}

			if readyNow := (detC.State == string(bridgeapi.StateReady)); readyNow {
				instanceObj.Status.Phase = crunchybridgev1alpha1.PhaseReady
				statusErr := r.updateStatus(instanceObj, metav1.ConditionTrue, Ready, InstanceSuccessMessage)
				if statusErr != nil {
					logger.Error(statusErr, "Error in updating CrunchyBridgeInstance status")
					return ctrl.Result{Requeue: true}, statusErr
				}
				logger.Info("cluster created", "name", instanceObj.Spec.Name)
			}

			if err := r.Status().Update(ctx, instanceObj); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true, RequeueAfter: WatchInt}, nil

		case crunchybridgev1alpha1.PhaseReady:
			// TODO: Monitor changes, change state machine (phoenix)

		default:
			return ctrl.Result{}, fmt.Errorf("unrecognized phase: %s", instanceObj.Status.Phase)
		}

	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CrunchyBridgeInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dbaasredhatcomv1alpha1.CrunchyBridgeInstance{}).
		Complete(r)
}

func listContains(list []string, s string) bool {
	for _, str := range list {
		if str == s {
			return true
		}
	}
	return false
}

func (r *CrunchyBridgeInstanceReconciler) createFromSpec(spec dbaasv1alpha1.DBaaSInstanceSpec, bridgeapiClient *bridgeapi.Client) (bridgeapi.CreateRequest, error) {
	req := bridgeapi.CreateRequest{
		Name:           spec.Name,
		PGMajorVersion: 13,
		Plan:           "trial",
		Provider:       spec.CloudProvider,
		Region:         spec.CloudRegion,
	}

	if teamID, ok := spec.OtherInstanceParams["TeamID"]; ok {
		req.TeamID = teamID
	} else {
		// Lookup TeamID
		if id, err := bridgeapiClient.DefaultTeamID(); err != nil {
			return req, err
		} else {
			req.TeamID = id
		}
	}

	if majorVersion, ok := spec.OtherInstanceParams["PGMajorVer"]; ok {
		req.PGMajorVersion = convertInt(majorVersion)
	}

	if plan, ok := spec.OtherInstanceParams["Plan"]; ok {
		req.Plan = plan
	}
	if storage, ok := spec.OtherInstanceParams["Storage"]; ok {
		req.StorageGB = convertInt(storage)
	}
	if isHA, ok := spec.OtherInstanceParams["HighAvail"]; ok {
		req.HighAvailability = convertBool(isHA)
	}

	// Treat "default" plan as a request for trial. This ends up ignoring
	// the requested Cloud* attributes and other attributes forced by the
	// trial configuration
	if req.Plan == "trial" {
		req.StorageGB = 10
		req.HighAvailability = false
		req.Plan = "hobby-2"
		req.Trial = true

		// Allow requesting region if requesting on AWS, where trials
		// are allowed, otherwise overwrite requested to trial-allowed
		if spec.CloudProvider != "aws" {
			req.Provider = "aws"
			req.Region = "us-east-1"
		}
	}

	return req, nil
}

func convertInt(input string) int {
	output, _ := strconv.Atoi(input)
	return output
}

func convertBool(input string) bool {
	output, _ := strconv.ParseBool(input)
	return output
}

// updateStatusFromDetail performs an update to the status of the API object
// if an error case is returned, it is assumed the status will not be
// written back to the server, so in-place changes will be lost
func (r *CrunchyBridgeInstanceReconciler) updateStatusFromDetail(
	det bridgeapi.ClusterDetail,
	statusObj *dbaasv1alpha1.DBaaSInstanceStatus) error {

	// Using ID field as a heuristic that a valid ClusterDetail was provided
	// and blindly updating the detail from there instead of per-field checks
	if det.ID == "" {
		return errors.New("received cluster detail with no ID")
	}
	statusObj.InstanceID = det.ID
	statusObj.InstanceInfo = map[string]string{
		CLUSTER_NAME:  det.Name,
		TEAM_ID:       det.TeamID,
		CPU:           strconv.Itoa(det.CPU),
		MEMORY:        strconv.Itoa(det.MemoryGB),
		STORAGE:       strconv.Itoa(det.StorageGB),
		MAJOR_VERSION: strconv.Itoa(det.PGMajorVersion),
		IS_HA:         strconv.FormatBool(det.HighAvailability),
		PROVIDER_ID:   det.ProviderID,
		REGION_ID:     det.RegionID,
	}

	return nil
}

// updateStatus
func (r *CrunchyBridgeInstanceReconciler) updateStatus(instanceObj *dbaasredhatcomv1alpha1.CrunchyBridgeInstance, conidtionStatus metav1.ConditionStatus, reason, message string) error {
	setStatusCondition(instanceObj, ProvisionReady, conidtionStatus, reason, message)
	if err := r.Client.Status().Update(context.Background(), instanceObj); err != nil {
		return err
	}
	return nil
}
