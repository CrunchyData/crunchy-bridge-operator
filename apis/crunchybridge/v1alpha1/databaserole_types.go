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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DatabaseRoleSpec defines the desired state of DatabaseRole
type DatabaseRoleSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// identifies the cluster on which this role exists
	ClusterID string `json:"cluster_id"`
	// identifies the requested role name, defaults to a system-generated
	// name if not provided
	// +optional
	RoleName string `json:"role_name"`
}

// DatabaseRoleStatus defines the observed state of DatabaseRole
type DatabaseRoleStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	// represents the creation state of the request
	Phase string `json:"phase"`
	// represents the creation time for the role
	Created string `json:"created_at"`
	// represents the role provisioned for this request
	RoleName string `json:"role_name"`
	// represents the secret associated with this role
	CredentialRef NamespacedName `json:"credential_ref"`
}

// Namespaced name is a light representation of a name in a namespace
type NamespacedName struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DatabaseRole is the Schema for the databaseroles API
type DatabaseRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseRoleSpec   `json:"spec,omitempty"`
	Status DatabaseRoleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DatabaseRoleList contains a list of DatabaseRole
type DatabaseRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DatabaseRole `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DatabaseRole{}, &DatabaseRoleList{})
}
