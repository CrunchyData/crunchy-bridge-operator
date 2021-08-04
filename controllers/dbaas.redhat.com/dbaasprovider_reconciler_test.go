package dbaasredhatcom

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const commonName = "test-provider"

var _ = Describe("DBaaS Provider Controller", func() {

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: commonName + "-cluster-role"},
		Rules:      []rbacv1.PolicyRule{{Verbs: []string{"create"}, APIGroups: []string{"dbaas.redhat.com"}, Resources: []string{"*"}}},
	}

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{GenerateName: commonName + "-"},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: commonName + "-cluster-role"},
		Subjects:   []rbacv1.Subject{{APIGroup: "rbac.authorization.k8s.io", Kind: "User", Name: commonName}},
	}

	BeforeEach(assertResourceCreation(clusterRole))
	BeforeEach(assertResourceCreation(clusterRoleBinding))
	AfterEach(assertResourceDeletion(clusterRole))
	AfterEach(assertResourceDeletion(clusterRoleBinding))

	Context(" provider creation", func() {
		It("Should create provider cr successfully", func() {

			dep := &v1.Deployment{}

			assertResourceCreation(dep)

			clusterRoleList := &rbacv1.ClusterRoleList{}
			err := k8sClient.List(context.Background(), clusterRoleList)

			Expect(err).NotTo(HaveOccurred())
			// check CRD exists
			CRDEventuallyExists("dbaasproviders.dbaas.redhat.com")

			By("creating a instance")
			providerCR := bridgeProviderCR(clusterRoleList)
			assertResourceCreation(providerCR)
			assertResourceDeletion(providerCR)

		})
	})

})
