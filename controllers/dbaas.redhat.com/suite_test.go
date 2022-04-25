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
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	label "k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	dbaasredhatcomv1alpha1 "github.com/CrunchyData/crunchy-bridge-operator/apis/dbaas.redhat.com/v1alpha1"
	dbaasv1alpha1 "github.com/RHEcosystemAppEng/dbaas-operator/api/v1alpha1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context

const (
	installNamespaceEnvVar = "INSTALL_NAMESPACE"
	testNamespace          = "default"
	operatorConditionName  = "OPERATOR_CONDITION_NAME"
	operatorConditionValue = "crunchy-bridge-operator.v0.0.1"
	crunchybridgeAPIURL    = "https://api.crunchybridge.com"
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
			filepath.Join("..", "..", "config", "test", "crd"),
		},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	clientset, err := kubernetes.NewForConfig(cfg)
	Expect(err).NotTo(HaveOccurred())
	Expect(clientset).NotTo(BeNil())

	err = dbaasredhatcomv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = dbaasredhatcomv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = dbaasv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = apiextv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = dbaasredhatcomv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	ctx = context.Background()

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		NewCache: cache.BuilderWithOptions(cache.Options{
			SelectorsByObject: cache.SelectorsByObject{
				&corev1.Secret{}: {
					Label: label.SelectorFromSet(label.Set{
						dbaasv1alpha1.TypeLabelKey: dbaasv1alpha1.TypeLabelValue,
					}),
				},
			},
		}),
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(mgr).ToNot(BeNil())

	err = os.Setenv(installNamespaceEnvVar, testNamespace)
	Expect(err).NotTo(HaveOccurred())

	err = os.Setenv(operatorConditionName, operatorConditionValue)
	Expect(err).NotTo(HaveOccurred())

	inventoryReconciler := &CrunchyBridgeInventoryReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		APIBaseURL: crunchybridgeAPIURL,
		Log:        ctrl.Log.WithName("controllers").WithName("CrunchyBridgeInventoryReconciler"),
	}
	err = inventoryReconciler.SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	DbaaSProviderReconciler := &DBaaSProviderReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Log:       ctrl.Log.WithName("controllers").WithName("DBaaSProviderReconciler"),
		Clientset: clientset,
	}
	err = DbaaSProviderReconciler.SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	dbaasredhatcomcontrollers := &CrunchyBridgeConnectionReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		Clientset:  clientset,
		APIBaseURL: crunchybridgeAPIURL,
	}
	err = dbaasredhatcomcontrollers.SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	dbaasInstance := &CrunchyBridgeInstanceReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		APIBaseURL: crunchybridgeAPIURL,
	}
	err = dbaasInstance.SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = mgr.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()

}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
