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

package main

import (
	"flag"
	"net/url"
	"os"
	"strings"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	crunchybridgev1alpha1 "github.com/CrunchyData/crunchy-bridge-operator/apis/crunchybridge/v1alpha1"
	crunchybridgecontrollers "github.com/CrunchyData/crunchy-bridge-operator/controllers/crunchybridge"

	//+kubebuilder:scaffold:imports
	"github.com/CrunchyData/crunchy-bridge-operator/internal/bridgeapi"
	"github.com/CrunchyData/crunchy-bridge-operator/internal/kubeadapter"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(crunchybridgev1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

type mainConfig struct {
	apiURL    string
	clientset *kubernetes.Clientset
}

var dbaasInit func(ctrl.Manager, mainConfig)

func main() {
	// Variables from boilerplate
	var metricsAddr, probeAddr string
	var enableLeaderElection bool

	var crunchybridgeAPIURL string
	// Namespace and Name for APIKey secret default values
	credNamespace := "default"
	credName := "crunchybridge_api_key"

	flag.StringVar(&crunchybridgeAPIURL, "crunchybridgeapi-url", "https://api.crunchybridge.com", "the Crunchy bridge API URL")
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Pull optional configuration details from env
	if cns, ok := os.LookupEnv("API_CRED_NS"); ok {
		credNamespace = cns
	}
	if cn, ok := os.LookupEnv("API_CRED_NAME"); ok {
		credName = cn
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "0b67260c.crunchydata.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	cfg := mgr.GetConfig()
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		setupLog.Error(err, "unable to create clientset")
		os.Exit(1)
	}

	crunchybridgeAPIURL = strings.TrimRight(crunchybridgeAPIURL, "/")

	apiURL, err := url.Parse(crunchybridgeAPIURL)
	if err != nil {
		setupLog.Error(err, "error parsing API URL", "URL", crunchybridgeAPIURL)
		os.Exit(1)
	}

	// Create client directly for querying non-managed object
	crClient, err := ctrlclient.New(mgr.GetConfig(), ctrlclient.Options{})
	if err != nil {
		setupLog.Error(err, "failed to init client to get api credentials")
		os.Exit(1)
	}

	// Initialize credential provider from environment
	ksp := &kubeadapter.KubeSecretCredentialProvider{
		Client:      crClient,
		Namespace:   credNamespace,
		Name:        credName,
		KeyField:    "api_key",
		SecretField: "api_secret",
	}

	bridgeClient, err := bridgeapi.NewClient(apiURL, ksp, bridgeapi.SetLogger(setupLog))
	if err != nil {
		setupLog.Error(err, "error setting up Crunchy Bridge API client")
		os.Exit(1)
	}

	// Set up manager with DBaaS controllers if built with option
	if dbaasInit != nil {
		dbaasInit(mgr, mainConfig{
			apiURL:    crunchybridgeAPIURL,
			clientset: clientset,
		})
	}

	if err = (&crunchybridgecontrollers.BridgeClusterReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		BridgeClient: bridgeClient,
		WatchInt:     10 * time.Second,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BridgeCluster")
		os.Exit(1)
	}
	if err = (&crunchybridgecontrollers.DatabaseRoleReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DatabaseRole")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
