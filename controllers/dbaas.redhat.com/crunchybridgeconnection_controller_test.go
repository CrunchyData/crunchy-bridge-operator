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
	"time"

	dbaasv1alpha1 "github.com/RHEcosystemAppEng/dbaas-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/CrunchyData/crunchy-bridge-operator/apis/dbaas.redhat.com/v1alpha1"
)

const (
	timeout          = time.Second * 10
	interval         = time.Millisecond * 250
	inventoryRefName = "test-inventory-ref"
)

var _ = Describe("CrunchyBridgeConnection controller", func() {

	Describe("CrunchyBridgeConnectionReconciler", func() {
		AfterEach(assertResourceDeletion(connectionCR()))

		Context("CrunchyBridgeConnection", func() {
			It("Should create connection instance successfully", func() {

				By("By creating inventory instance")

				inventory := createInventories(inventoryRefName)
				EventuallyExists(inventory)

				By("By creating connection instance")
				connection := connectionCR()
				Expect(k8sClient.Create(ctx, connection)).Should(Succeed())

				By("getting connection instance")
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(connection), connection)
				Expect(err).NotTo(HaveOccurred())

				By("getting a inventory instance")
				err = k8sClient.Get(ctx, types.NamespacedName{Namespace: connection.Spec.InventoryRef.Namespace, Name: connection.Spec.InventoryRef.Name}, inventory)
				Expect(err).NotTo(HaveOccurred())

			})

		})
	})
})

func connectionCR() *v1alpha1.CrunchyBridgeConnection {
	connectionName := "test-connection"
	instanceID := "testInstanceID"
	DBaaSConnectionSpec := &dbaasv1alpha1.DBaaSConnectionSpec{
		InventoryRef: dbaasv1alpha1.NamespacedName{
			Name:      inventoryRefName,
			Namespace: testNamespace,
		},
		InstanceID: instanceID,
	}
	connection := &v1alpha1.CrunchyBridgeConnection{
		ObjectMeta: metav1.ObjectMeta{
			Name:      connectionName,
			Namespace: testNamespace,
		},
		Spec: *DBaaSConnectionSpec,
	}

	return connection
}
