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
	dbaasredhatcom "github.com/CrunchyData/crunchy-bridge-operator/controllers/dbaas.redhat.com"

	"github.com/CrunchyData/crunchy-bridge-operator/controllers"
	"github.com/CrunchyData/crunchy-bridge-operator/internal/bridgeapi"
	"github.com/CrunchyData/crunchy-bridge-operator/internal/kubeadapter"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/url"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	crunchybridgev1alpha1 "github.com/CrunchyData/crunchy-bridge-operator/apis/crunchybridge/v1alpha1"
)

// BridgeTeamReconciler reconciles a BridgeTeam object
type BridgeTeamReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	APIBaseURL string
}

//+kubebuilder:rbac:groups=crunchybridge.crunchydata.com,resources=bridgeteams,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=crunchybridge.crunchydata.com,resources=bridgeteams/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=crunchybridge.crunchydata.com,resources=bridgeteams/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *BridgeTeamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "BridgeTeamReconciler", req.NamespacedName)

	var team crunchybridgev1alpha1.BridgeTeam
	err := r.Get(ctx, req.NamespacedName, &team)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// CR deleted since request queued, child objects getting GC'd, no requeue
			logger.Info("BridgeTeam resource not found, has been deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Error fetching BridgeTeam for reconcile")
		return ctrl.Result{}, err
	}
	bridgeapiClient, err := setupClient(r.Client, team, r.APIBaseURL, logger)
	if err != nil {
		statusErr := r.updateStatus(ctx, team, metav1.ConditionFalse, controllers.AuthenticationError, err.Error())
		if statusErr != nil {
			logger.Error(statusErr, "Error in updating BridgeTeam status")
			return ctrl.Result{Requeue: true}, statusErr
		}
		logger.Error(err, "Error while setting up CrunchyBridge Client")
		return ctrl.Result{}, err
	}
	logger.Info("Crunchy Bridge Client Configured ")
	err = r.discoverTeams(&team, bridgeapiClient, logger)
	if err != nil {
		statusErr := r.updateStatus(ctx, team, metav1.ConditionFalse, controllers.BackendError, err.Error())
		if statusErr != nil {
			logger.Error(statusErr, "Error in updating BridgeTeam status")
			return ctrl.Result{Requeue: true}, statusErr
		}
		logger.Error(err, "Error while querying the Teams from CrunchyBridge")
		return ctrl.Result{}, err
	}
	statusErr := r.updateStatus(ctx, team, metav1.ConditionTrue, controllers.SyncOK, controllers.SuccessTeam)
	if statusErr != nil {
		logger.Error(statusErr, "Error in updating BridgeTeam status")
		return ctrl.Result{Requeue: true}, statusErr
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BridgeTeamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&crunchybridgev1alpha1.BridgeTeam{}).
		Complete(r)
}

// updateStatus
func (r *BridgeTeamReconciler) updateStatus(ctx context.Context, team crunchybridgev1alpha1.BridgeTeam, conidtionStatus metav1.ConditionStatus, reason, message string) error {
	controllers.SetStatusCondition(&team, controllers.SpecSynced, conidtionStatus, reason, message)
	if err := r.Status().Update(ctx, &team); err != nil {
		return err
	}
	return nil
}

// discoverTeams query crunchy bridge and return list of teams
func (r *BridgeTeamReconciler) discoverTeams(team *crunchybridgev1alpha1.BridgeTeam, bridgeapi *bridgeapi.Client, logger logr.Logger) error {
	var Teams []crunchybridgev1alpha1.Team
	teamList, teamListErr := bridgeapi.ListAllTeams()
	if teamListErr != nil {
		logger.Error(teamListErr, "Error Listing the teams")
		return teamListErr
	}
	for _, team := range teamList.Teams {

		if team.IsPersonal {
			team.Name = "Personal"
		}

		Team := crunchybridgev1alpha1.Team{
			ID:   team.ID,
			Name: team.Name,
		}
		Teams = append(Teams, Team)
	}

	team.Status.Teams = Teams

	return nil
}

// setupClient
func setupClient(client client.Client, team crunchybridgev1alpha1.BridgeTeam, APIBaseURL string, logger logr.Logger) (*bridgeapi.Client, error) {
	baseUrl, err := url.Parse(APIBaseURL)
	if err != nil {
		logger.Error(err, "Malformed URL", "URL", APIBaseURL)
		return nil, err
	}
	kubeSecretProvider := &kubeadapter.KubeSecretCredentialProvider{
		Client:      client,
		Namespace:   team.Spec.CredentialsRef.Namespace,
		Name:        team.Spec.CredentialsRef.Name,
		KeyField:    dbaasredhatcom.KEYFIELDNAME,
		SecretField: dbaasredhatcom.SECRETFIELDNAME,
	}

	return bridgeapi.NewClient(baseUrl, kubeSecretProvider, bridgeapi.SetLogger(logger))
}
