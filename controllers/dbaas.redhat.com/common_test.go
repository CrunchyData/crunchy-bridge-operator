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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EventuallyExists checks if an object with the given namespace+name and type eventually exists.
func EventuallyExists(obj client.Object) {
	Eventually(func() bool {

		err := k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), obj)
		if errors.IsNotFound(err) {
			return false
		}
		Expect(err).NotTo(HaveOccurred())
		return true
	}).Should(BeTrue())
}

// CRDEventuallyExists checks if a custom resource definition with the given name eventually exists.
func CRDEventuallyExists(crdName string) {
	crd := &apiextv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: crdName,
		},
	}
	EventuallyExists(crd)
}

// assertResourceCreation create the resources
func assertResourceCreation(object client.Object) func() {
	return func() {
		By("creating resource")
		object.SetResourceVersion("")
		Expect(k8sClient.Create(ctx, object)).Should(Succeed())

		By("checking the resource created")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(object), object)
			if err != nil {
				return false
			}
			return true
		}, timeout, interval).Should(BeTrue())
	}
}

// assertResourceDeletion resource deletion
func assertResourceDeletion(object client.Object) func() {
	return func() {
		By("deleting resource")
		Expect(k8sClient.Delete(ctx, object)).Should(Succeed())

		By("checking the resource deleted")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(object), object)
			if err != nil && errors.IsNotFound(err) {
				return true
			}
			return false
		}, timeout, interval).Should(BeTrue())
	}
}
