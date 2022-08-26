/*
 * Copyright (C) 2020 Red Hat, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dbaasredhatcom

import (
	"context"
	"fmt"
	"os"
	"time"

	dbaasoperator "github.com/RHEcosystemAppEng/dbaas-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/apps/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	label "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	PROVIDER           = "Red Hat DBaaS / Crunchy Bridge"
	DISPLAYNAME        = "Crunchy Bridge managed PostgreSQL"
	DISPLAYDESCRIPTION = "The managed PostgreSQL database service that allows you to focus on your application, not your database. Harness the power of Postgres on the cloud provider of your choice with trusted Crunchy Data support."

	INVENTORYDATAVALUE     = "CrunchyBridgeInventory"
	CONNECTIONDATAVALUE    = "CrunchyBridgeConnection"
	INSTANCEDATAVALUE      = "CrunchyBridgeInstance"
	NAME                   = "crunchy-bridge-registration"
	ICONDATA               = "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"
	MEDIATYPE              = "image/png"
	KEYFIELDNAME           = "publicApiKey"
	KEYFIELDDISPLAYNAME    = "Public API Key"
	KEYFIELDHELPTEXT       = "The Public API Key is the Application ID value associated with your Crunchy Bridge Cloud account."
	SECRETFIELDNAME        = "privateApiSecret"
	SECRETFIELDDISPLAYNAME = "Private API Secret"
	SECRETFIELDHELPTEXT    = "The Private API Secret is the Application Secret associated with your Crunchy Bridge Cloud account."
	RELATEDTOLABELNAME     = "related-to"
	RELATEDTOLABELVALUE    = "dbaas-operator"
	TYPELABELNAME          = "type"
	TYPELABELVALUE         = "dbaas-provider-registration"
	DBAASPROVIDERKIND      = "DBaaSProvider"
	PROVISION_DOC_URL      = "https://docs.crunchybridge.com/quickstart/provision"
	PROVISION_DESCRIPTION  = "Crunchy Bridge by Crunchy Data offers free trial instances through RHODA. To provision a trial instance, provision through RHODA using default parameters or specify the plan as 'trial'. For further information on provisioning paid instances via the Crunchy Bridge platform, please refer to our provisioning documentation."
)

var labels = map[string]string{RELATEDTOLABELNAME: RELATEDTOLABELVALUE, TYPELABELNAME: TYPELABELVALUE}

type DBaaSProviderReconciler struct {
	client.Client
	*runtime.Scheme
	Log                      logr.Logger
	Clientset                *kubernetes.Clientset
	operatorNameVersion      string
	operatorInstallNamespace string
}

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch
// +kubebuilder:rbac:groups=dbaas.redhat.com,resources=dbaasproviders,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dbaas.redhat.com,resources=dbaasproviders/status,verbs=get;update;patch

func (r *DBaaSProviderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := r.Log.WithValues("during", "DBaaSProvider Reconciler")

	// due to predicate filtering, we'll only reconcile this operator's own deployment when it's seen the first time
	// meaning we have a reconcile entry-point on operator start-up, so now we can create a cluster-scoped resource
	// owned by the operator's ClusterRole to ensure cleanup on uninstall

	dep := &v1.Deployment{}
	if err := r.Get(ctx, req.NamespacedName, dep); err != nil {
		if errors.IsNotFound(err) {
			// CR deleted since request queued, child objects getting GC'd, no requeue
			log.Info("deployment not found, deleted, no requeue")
			return ctrl.Result{}, nil
		}
		// error fetching deployment, requeue and try again
		log.Error(err, "error fetching Deployment CR")
		return ctrl.Result{}, err
	}

	isCrdInstalled, err := r.checkCrdInstalled(dbaasoperator.GroupVersion.String(), DBAASPROVIDERKIND)
	if err != nil {
		log.Error(err, "error discovering GVK")
		return ctrl.Result{}, err
	}
	if !isCrdInstalled {
		log.Info("CRD not found, requeueing with rate limiter")
		// returning with 'Requeue: true' will invoke our custom rate limiter seen in SetupWithManager below
		return ctrl.Result{Requeue: true}, nil
	}

	instance := &dbaasoperator.DBaaSProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: NAME,
		},
	}
	if err := r.Get(ctx, client.ObjectKeyFromObject(instance), instance); err != nil {
		if errors.IsNotFound(err) {
			// CR deleted since request queued, child objects getting GC'd, no requeue
			log.Info("resource not found, creating now")

			// crunchy bridge registration custom resource isn't present,so create now with ClusterRole owner for GC
			opts := &client.ListOptions{
				LabelSelector: label.SelectorFromSet(map[string]string{
					"olm.owner":      r.operatorNameVersion,
					"olm.owner.kind": "ClusterServiceVersion",
				}),
			}
			clusterRoleList := &rbac.ClusterRoleList{}
			if err := r.List(context.Background(), clusterRoleList, opts); err != nil {
				log.Error(err, "unable to list ClusterRoles to seek potential operand owners")
				return ctrl.Result{}, err
			}

			if len(clusterRoleList.Items) < 1 {
				err := errors.NewNotFound(
					schema.GroupResource{Group: "rbac.authorization.k8s.io", Resource: "ClusterRole"}, "potentialOwner")
				log.Error(err, "could not find ClusterRole owned by CSV to inherit operand")
				return ctrl.Result{}, err
			}

			instance = bridgeProviderCR(clusterRoleList)
			if err := r.Create(ctx, instance); err != nil {
				log.Error(err, "error while creating new cluster-scoped resource")
				return ctrl.Result{}, err
			} else {
				log.Info("cluster-scoped resource created")
				return ctrl.Result{}, nil
			}
		}
		// error fetching the resource, requeue and try again
		log.Error(err, "error fetching the resource")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// bridgeProviderCR CR for crunchy bridge registration
func bridgeProviderCR(clusterRoleList *rbac.ClusterRoleList) *dbaasoperator.DBaaSProvider {
	instance := &dbaasoperator.DBaaSProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: NAME,
			OwnerReferences: []metav1.OwnerReference{
				{

					APIVersion:         "rbac.authorization.k8s.io/v1",
					Kind:               "ClusterRole",
					UID:                clusterRoleList.Items[0].GetUID(), // doesn't really matter which 'item' we use
					Name:               clusterRoleList.Items[0].Name,
					Controller:         pointer.BoolPtr(true),
					BlockOwnerDeletion: pointer.BoolPtr(false),
				},
			},
			Labels: labels,
		},

		Spec: dbaasoperator.DBaaSProviderSpec{
			Provider: dbaasoperator.DatabaseProvider{
				Name:               PROVIDER,
				DisplayName:        DISPLAYNAME,
				DisplayDescription: DISPLAYDESCRIPTION,
				Icon: dbaasoperator.ProviderIcon{
					Data:      ICONDATA,
					MediaType: MEDIATYPE,
				},
			},
			InventoryKind:  INVENTORYDATAVALUE,
			ConnectionKind: CONNECTIONDATAVALUE,
			InstanceKind:   INSTANCEDATAVALUE,
			CredentialFields: []dbaasoperator.CredentialField{
				{
					Key:         KEYFIELDNAME,
					DisplayName: KEYFIELDDISPLAYNAME,
					Type:        "string",
					Required:    true,
					HelpText:    KEYFIELDHELPTEXT,
				},
				{
					Key:         SECRETFIELDNAME,
					DisplayName: SECRETFIELDDISPLAYNAME,
					Type:        "maskedstring",
					Required:    true,
					HelpText:    SECRETFIELDHELPTEXT,
				},
			},
			AllowsFreeTrial:              true,
			ExternalProvisionURL:         PROVISION_DOC_URL,
			ExternalProvisionDescription: PROVISION_DESCRIPTION,
			InstanceParameterSpecs: []dbaasoperator.InstanceParameterSpec{
				{
					Name:        "Name",
					DisplayName: "Cluster Name",
					Type:        "string",
					Required:    true,
				},
				{
					Name:        "TeamID",
					DisplayName: "Team ID",
					Type:        "string",
					Required:    false,
				},
				{
					Name:         "PGMajorVer",
					DisplayName:  "Version",
					Type:         "int",
					Required:     true,
					DefaultValue: "13",
				},
				{
					Name:         "Provider",
					DisplayName:  "Cloud Service Provider",
					Type:         "string",
					Required:     true,
					DefaultValue: "aws",
				},
				{
					Name:         "Region",
					DisplayName:  "Region",
					Type:         "string",
					Required:     true,
					DefaultValue: "us-east-1",
				},
				{
					Name:         "Plan",
					DisplayName:  "Plan",
					Type:         "string",
					Required:     true,
					DefaultValue: "hobby-2",
				},
				{
					Name:         "Storage",
					DisplayName:  "Storage",
					Type:         "int",
					Required:     true,
					DefaultValue: "10",
				},
				{
					Name:        "HighAvail",
					DisplayName: "High Availability",
					Type:        "bool",
					Required:    false,
				},
			},
		},
	}
	return instance
}

// CheckCrdInstalled checks whether dbaas provider CRD, has been created yet
func (r *DBaaSProviderReconciler) checkCrdInstalled(groupVersion, kind string) (bool, error) {
	resources, err := r.Clientset.Discovery().ServerResourcesForGroupVersion(groupVersion)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	for _, r := range resources.APIResources {
		if r.Kind == kind {
			return true, nil
		}
	}
	return false, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DBaaSProviderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	log := r.Log.WithValues("during", "DBaaSProviderReconciler setup")

	// envVar set in controller-manager's Deployment YAML
	if operatorInstallNamespace, found := os.LookupEnv("INSTALL_NAMESPACE"); !found {
		err := fmt.Errorf("INSTALL_NAMESPACE must be set")
		log.Error(err, "error fetching envVar")
		return err
	} else {
		r.operatorInstallNamespace = operatorInstallNamespace
	}

	// envVar set for all operators
	if operatorNameEnvVar, found := os.LookupEnv("OPERATOR_CONDITION_NAME"); !found {
		err := fmt.Errorf("OPERATOR_CONDITION_NAME must be set")
		log.Error(err, "error fetching envVar")
		return err
	} else {
		r.operatorNameVersion = operatorNameEnvVar
	}

	customRateLimiter := workqueue.NewItemExponentialFailureRateLimiter(30*time.Second, 30*time.Minute)

	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{RateLimiter: customRateLimiter}).
		For(
			&v1.Deployment{},
			builder.WithPredicates(r.ignoreOtherDeployments()),
			builder.OnlyMetadata,
		).
		Complete(r)
}

//ignoreOtherDeployments  only on a 'create' event is issued for the deployment
func (r *DBaaSProviderReconciler) ignoreOtherDeployments() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return r.evaluatePredicateObject(e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

func (r *DBaaSProviderReconciler) evaluatePredicateObject(obj client.Object) bool {
	lbls := obj.GetLabels()
	if obj.GetNamespace() == r.operatorInstallNamespace {
		if val, keyFound := lbls["olm.owner.kind"]; keyFound {
			if val == "ClusterServiceVersion" {
				if val, keyFound := lbls["olm.owner"]; keyFound {
					return val == r.operatorNameVersion
				}
			}
		}
	}
	return false
}
