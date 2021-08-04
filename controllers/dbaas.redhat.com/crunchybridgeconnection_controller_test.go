package dbaasredhatcom

import (
	"github.com/CrunchyData/crunchy-bridge-operator/apis/dbaas.redhat.com/v1alpha1"
	dbaasv1alpha1 "github.com/RHEcosystemAppEng/dbaas-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	timeout          = time.Second * 10
	duration         = time.Second * 10
	interval         = time.Millisecond * 250
	inventoryRefName = "test-inventory-ref"
)

var _ = Describe("CrunchyBridgeConnection controller", func() {

	Describe("CrunchyBridgeConnectionReconciler", func() {
		BeforeEach(func() {})
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
