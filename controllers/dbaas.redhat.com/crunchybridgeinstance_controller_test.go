package dbaasredhatcom

import (
	dbaasv1alpha1 "github.com/RHEcosystemAppEng/dbaas-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/CrunchyData/crunchy-bridge-operator/apis/dbaas.redhat.com/v1alpha1"

	"time"
)

var _ = Describe("CrunchyBridgeInstance controller ", func() {
	Context("CrunchyBridgeInstance", func() {
		It("Should create and delete bridge instance cr successfully", func() {

			By("creating first inventory")
			inventory := createInventories(inventoryName)
			cond := GetInventoryCondition(inventory, string(SpecSynced))
			Expect(cond.Status).Should(Equal(metav1.ConditionTrue))

			By("create the bridge instance")
			instance := createInstance(inventoryName)
			cond = GetIInstanceCondition(instance, string(ProvisionReady))
			Expect(cond.Status).Should(Equal(metav1.ConditionTrue))

			By("deleting instance")
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())

			By("checking the instance deleted")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
				if err != nil && errors.IsNotFound(err) {
					return true
				}
				return false
			}, timeout, interval).Should(BeTrue())

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

func createInstance(inventoryName string) *v1alpha1.CrunchyBridgeInstance {

	DBaaSInstanceSpec := &dbaasv1alpha1.DBaaSInstanceSpec{
		InventoryRef: dbaasv1alpha1.NamespacedName{
			Name:      inventoryName,
			Namespace: testNamespace,
		},
		Name:          "test-instance",
		CloudProvider: "aws",
		CloudRegion:   "test-region",
		OtherInstanceParams: map[string]string{
			"testParam": "test-param",
		},
	}
	instance := &v1alpha1.CrunchyBridgeInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      inventoryName,
			Namespace: testNamespace,
		},
		Spec: *DBaaSInstanceSpec,
	}
	By("creating a bridge instance")
	Expect(k8sClient.Create(ctx, instance)).To(Succeed())

	updateMockInstanceStatus(instance)

	return instance
}

func updateMockInstanceStatus(instance *v1alpha1.CrunchyBridgeInstance) {
	By("setting up status")
	lastTransitionTime, err := time.Parse(time.RFC3339, "2021-06-30T22:17:55-04:00")
	Expect(err).NotTo(HaveOccurred())

	lastTransitionTime = lastTransitionTime.In(time.Local)
	status := &dbaasv1alpha1.DBaaSInstanceStatus{
		InstanceID: "testInstanceID",
		InstanceInfo: map[string]string{
			"testInstanceInfo": "testInstanceInfo",
		},
		Phase: "Ready",
		Conditions: []metav1.Condition{
			{
				Type:               ProvisionReady,
				Status:             metav1.ConditionTrue,
				Reason:             Ready,
				LastTransitionTime: metav1.Time{Time: lastTransitionTime},
			},
		},
	}
	instance.Status = *status
	Expect(k8sClient.Status().Update(ctx, instance)).Should(Succeed())

}
