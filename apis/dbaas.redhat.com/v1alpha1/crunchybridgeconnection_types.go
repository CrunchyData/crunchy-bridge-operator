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

package v1alpha1

import (
	dbaasv1alpha1 "github.com/RHEcosystemAppEng/dbaas-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CrunchyBridgeConnection is the Schema for the crunchybridgeconnections API
type CrunchyBridgeConnection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   dbaasv1alpha1.DBaaSConnectionSpec   `json:"spec,omitempty"`
	Status dbaasv1alpha1.DBaaSConnectionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CrunchyBridgeConnectionList contains a list of CrunchyBridgeConnection
type CrunchyBridgeConnectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CrunchyBridgeConnection `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CrunchyBridgeConnection{}, &CrunchyBridgeConnectionList{})
}
