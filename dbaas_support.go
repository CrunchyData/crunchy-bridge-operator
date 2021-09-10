// +build dbaas

package main

import (
	"os"

	dbaasredhatcomv1alpha1 "github.com/CrunchyData/crunchy-bridge-operator/apis/dbaas.redhat.com/v1alpha1"
	dbaasredhatcomcontrollers "github.com/CrunchyData/crunchy-bridge-operator/controllers/dbaas.redhat.com"
	dbaasoperator "github.com/RHEcosystemAppEng/dbaas-operator/api/v1alpha1"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func init() {
	utilruntime.Must(dbaasredhatcomv1alpha1.AddToScheme(scheme))
	utilruntime.Must(dbaasoperator.AddToScheme(scheme))

	dbaasInit = enableDBaaSExtension
}

func enableDBaaSExtension(mgr ctrl.Manager, cfg mainConfig) {
	inventoryReconciler := &dbaasredhatcomcontrollers.CrunchyBridgeInventoryReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		APIBaseURL: cfg.apiURL,
	}

	if err := inventoryReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CrunchyBridgeInventory")
		os.Exit(1)
	}

	dbaaSProviderReconciler := &dbaasredhatcomcontrollers.DBaaSProviderReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Log:       setupLog,
		Clientset: cfg.clientset,
	}

	if err := dbaaSProviderReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DBaaSProvider")
		os.Exit(1)
	}

	if err := (&dbaasredhatcomcontrollers.CrunchyBridgeConnectionReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		Clientset:  cfg.clientset,
		APIBaseURL: cfg.apiURL,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CrunchyBridgeConnection")
		os.Exit(1)
	}

}
