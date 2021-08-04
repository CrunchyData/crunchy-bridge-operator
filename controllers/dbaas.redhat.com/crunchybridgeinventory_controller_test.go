package dbaasredhatcom

import (
	"github.com/CrunchyData/crunchy-bridge-operator/apis/dbaas.redhat.com/v1alpha1"
	dbaasv1alpha1 "github.com/RHEcosystemAppEng/dbaas-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	credentialsRefName = "test-credentials-ref"
	inventoryName      = "test-inventory"
	privateApiSecret   = "test"
	publicApiKey       = "test"
)

var _ = Describe("CrunchyBridgeInventory controller", func() {

	BeforeEach(func() {})
	AfterEach(func() {})
	Context("CrunchyBridgeInventories", func() {
		It("Should create, update and delete inventory cr successfully", func() {

			By("creating inventory and updating status")
			inventory := createInventories(inventoryName)

			cond := GetInventoryCondition(inventory, string(SpecSynced))
			Expect(cond.Status).Should(Equal(metav1.ConditionTrue))

			By("deleting inventory")
			Expect(k8sClient.Delete(ctx, inventory)).Should(Succeed())

			By("checking the inventory deleted")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(inventory), inventory)
				if err != nil && errors.IsNotFound(err) {
					return true
				}
				return false
			}, timeout, interval).Should(BeTrue())
		})
	})

})

func updateMockStatus(inventory *v1alpha1.CrunchyBridgeInventory) {
	By("setting up status")
	lastTransitionTime, err := time.Parse(time.RFC3339, "2021-06-30T22:17:55-04:00")
	Expect(err).NotTo(HaveOccurred())

	lastTransitionTime = lastTransitionTime.In(time.Local)
	status := &dbaasv1alpha1.DBaaSInventoryStatus{
		Instances: []dbaasv1alpha1.Instance{
			{
				InstanceID: "testInstanceID",
				Name:       "testInstance",
				InstanceInfo: map[string]string{
					"testInstanceInfo": "testInstanceInfo",
				},
			},
		},
		Conditions: []metav1.Condition{
			{
				Type:               "SpecSynced",
				Status:             metav1.ConditionTrue,
				Reason:             "SyncOK",
				LastTransitionTime: metav1.Time{Time: lastTransitionTime},
			},
		},
	}
	inventory.Status = *status
	Expect(k8sClient.Status().Update(ctx, inventory)).Should(Succeed())

}

func createInventories(inventoryName string) *v1alpha1.CrunchyBridgeInventory {
	credentialSecret := createSecret(testNamespace)

	DBaaSInventorySpec := &dbaasv1alpha1.DBaaSInventorySpec{
		CredentialsRef: &dbaasv1alpha1.NamespacedName{
			Name:      credentialSecret.Name,
			Namespace: testNamespace,
		},
	}
	inventory := &v1alpha1.CrunchyBridgeInventory{
		ObjectMeta: metav1.ObjectMeta{
			Name:      inventoryName,
			Namespace: testNamespace,
		},
		Spec: *DBaaSInventorySpec,
	}
	By("creating a inventory instance")
	Expect(k8sClient.Create(ctx, inventory)).To(Succeed())

	updateMockStatus(inventory)

	return inventory
}

func createSecret(namespace string) *corev1.Secret {
	credentialSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: credentialsRefName + "-",
			Namespace:    namespace,
		},
		Data: map[string][]byte{
			KEYFIELDNAME:    []byte(publicApiKey),
			SECRETFIELDNAME: []byte(privateApiSecret),
		},
	}
	Expect(k8sClient.Create(ctx, credentialSecret)).To(Succeed())
	return credentialSecret
}
