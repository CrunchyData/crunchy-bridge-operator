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

package crunchybridge

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	crunchybridgev1alpha1 "github.com/CrunchyData/crunchy-bridge-operator/apis/crunchybridge/v1alpha1"
	"github.com/CrunchyData/crunchy-bridge-operator/internal/bridgeapi"
)

const (
	bcFinalizer = "crunchybridge.com/bridgecluster-finalizer"
)

// BridgeClusterReconciler reconciles a BridgeCluster object
type BridgeClusterReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	BridgeClient *bridgeapi.Client
	WatchInt     time.Duration
}

//+kubebuilder:rbac:groups=crunchybridge.crunchydata.com,resources=bridgeclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=crunchybridge.crunchydata.com,resources=bridgeclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=crunchybridge.crunchydata.com,resources=bridgeclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *BridgeClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "BridgeCluster", req.NamespacedName)

	if r.BridgeClient == nil {
		err := errors.New("Uninitialized client")
		logger.Error(err, "No CrunchyBridge client configured")
		return ctrl.Result{}, err
	}

	clusterObj := &crunchybridgev1alpha1.BridgeCluster{}
	if err := r.Get(ctx, req.NamespacedName, clusterObj); err != nil {
		if apierrors.IsNotFound(err) {
			// Likely deleted before action or extra pass post-deletion, no-op
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Error fetching BridgeCluster object for reconciliation")
		return ctrl.Result{}, err
	}

	if clusterObj.DeletionTimestamp != nil && !clusterObj.DeletionTimestamp.IsZero() {
		// Cluster deletion request / process finalizer
		if listContains(clusterObj.Finalizers, bcFinalizer) {
			if id := clusterObj.Status.Cluster.ID; id != "" {
				if err := r.BridgeClient.DeleteCluster(id); err != nil {
					return ctrl.Result{}, err
				}
			}
			controllerutil.RemoveFinalizer(clusterObj, bcFinalizer)
			if err := r.Update(ctx, clusterObj); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		switch clusterObj.Status.Phase {
		case crunchybridgev1alpha1.PhaseUnknown:
			fmt.Printf("Phase: Unknown\n")
			// New object, add our finalizer
			if !listContains(clusterObj.Finalizers, bcFinalizer) {
				controllerutil.AddFinalizer(clusterObj, bcFinalizer)
				if err := r.Update(ctx, clusterObj); err != nil {
					return ctrl.Result{}, err
				}
			}

			// Set pending phase after so any errors in setting finalizer
			// don't advance state
			clusterObj.Status.Phase = crunchybridgev1alpha1.PhasePending
			if err := r.Status().Update(ctx, clusterObj); err != nil {
				return ctrl.Result{}, err
			}

		case crunchybridgev1alpha1.PhasePending:
			fmt.Printf("Phase: Pending\n")
			req, err := r.createFromSpec(clusterObj.Spec)
			if err != nil {
				return ctrl.Result{}, err
			}

			if err := r.BridgeClient.CreateCluster(req); err != nil {
				return ctrl.Result{}, err
			}

			// Assuming the request was sent, update phase
			clusterObj.Status.Phase = crunchybridgev1alpha1.PhaseCreating
			clusterObj.Status.Updated = time.Now().Format(time.RFC3339)
			if err := r.Status().Update(ctx, clusterObj); err != nil {
				return ctrl.Result{}, err
			}

		case crunchybridgev1alpha1.PhaseCreating:
			fmt.Printf("Phase: %s - ver: %s\n", clusterObj.Status.Phase, clusterObj.ResourceVersion)
			var detC bridgeapi.ClusterDetail
			if cid := clusterObj.Status.Cluster.ID; cid == "" {
				c, err := r.BridgeClient.ClusterByName(clusterObj.Spec.Name)
				if err != nil {
					return ctrl.Result{}, err
				}
				detC = c
			} else {
				c, err := r.BridgeClient.ClusterDetail(cid)
				if err != nil {
					return ctrl.Result{}, err
				}
				detC = c
			}

			if err := r.updateStatusFromDetail(detC, &clusterObj.Status); err != nil {
				return ctrl.Result{}, err
			}

			if readyNow := (detC.State == string(bridgeapi.StateReady)); readyNow {
				clusterObj.Status.Phase = crunchybridgev1alpha1.PhaseReady
			}

			clusterObj.Status.Updated = time.Now().Format(time.RFC3339)
			if err := r.Status().Update(ctx, clusterObj); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true, RequeueAfter: r.WatchInt}, nil

		case crunchybridgev1alpha1.PhaseReady:
			fmt.Printf("Phase: Ready\n")
			// TODO: Monitor changes, change state machine (phoenix)

		default:
			return ctrl.Result{}, fmt.Errorf("unrecognized phase: %s", clusterObj.Status.Phase)
		}

	}

	// Perform create
	// Set status info

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BridgeClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&crunchybridgev1alpha1.BridgeCluster{}).
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

func (r *BridgeClusterReconciler) createFromSpec(spec crunchybridgev1alpha1.BridgeClusterSpec) (bridgeapi.CreateRequest, error) {
	req := bridgeapi.CreateRequest{
		Name:             spec.Name,
		TeamID:           spec.TeamID,
		Plan:             spec.Plan,
		StorageMB:        spec.StorageMB,
		Provider:         spec.Provider,
		Region:           spec.Region,
		PGMajorVersion:   spec.PGMajorVer,
		HighAvailability: spec.HighAvail,
	}

	if tid := spec.TeamID; tid == "" {
		// Lookup TeamID
		if id, err := r.BridgeClient.PersonalTeamID(); err != nil {
			return req, err
		} else {
			req.TeamID = id
		}
	}

	return req, nil
}

// updateStatusFromDetail performs an update to the status of the API object
// if an error case is returned, it is assumed the status will not be
// written back to the server, so in-place changes will be lost
func (r *BridgeClusterReconciler) updateStatusFromDetail(
	det bridgeapi.ClusterDetail,
	statusObj *crunchybridgev1alpha1.BridgeClusterStatus) error {

	// Using ID field as a heuristic that a valid ClusterDetail was provided
	// and blindly updating the detail from there instead of per-field checks
	if det.ID == "" {
		return errors.New("received cluster detail with no ID")
	}

	// Who
	statusObj.Cluster.ID = det.ID
	statusObj.Cluster.Name = det.Name
	statusObj.Cluster.TeamID = det.TeamID
	// What
	statusObj.Cluster.CPU = det.CPU
	statusObj.Cluster.MemoryGB = det.MemoryGB
	statusObj.Cluster.StorageMB = det.StorageMB
	statusObj.Cluster.PGMajorVer = det.PGMajorVersion
	statusObj.Cluster.HighAvail = det.HighAvailability
	// When
	statusObj.Cluster.Created = det.Created.Format(time.RFC3339)
	statusObj.Cluster.Updated = det.Created.Format(time.RFC3339)
	// Where
	statusObj.Cluster.ProviderID = det.ProviderID
	statusObj.Cluster.RegionID = det.RegionID

	if role, err := r.BridgeClient.DefaultConnRole(det.ID); err != nil {
		fmt.Printf("Unable to get connection role: %v\n", err)
	} else {
		dbURL, err := url.Parse(role.URI)
		if err != nil {
			return err
		}
		dbURL.User = nil
		statusObj.Connect.URI = dbURL.String()
		statusObj.Connect.ParentDBRole = role.Name
		statusObj.Connect.DatabaseName = strings.TrimLeft(dbURL.Path, "/")
	}

	statusObj.Updated = time.Now().Format(time.RFC3339)
	return nil
}
