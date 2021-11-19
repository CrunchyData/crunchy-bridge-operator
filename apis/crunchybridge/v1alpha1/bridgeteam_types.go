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

// BridgeTeamSpec defines the desired state of BridgeTeam
type BridgeTeamSpec struct {
	CredentialsRef *NamespacedName `json:"credentialsRef"`
}

// BridgeTeamStatus defines the observed state of BridgeTeam
type BridgeTeamStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// A list of teams associated to user
	Teams []Team `json:"teams,omitempty"`
}

type Team struct {

	// represents the ID of the team which user added
	ID string `json:"id"`

	// represents the name of the team
	Name string `json:"name,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BridgeTeam is the Schema for the bridgeteams API
type BridgeTeam struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BridgeTeamSpec   `json:"spec,omitempty"`
	Status BridgeTeamStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BridgeTeamList contains a list of BridgeTeam
type BridgeTeamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BridgeTeam `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BridgeTeam{}, &BridgeTeamList{})
}

// GetStatusConditions gets the status conditions from the
// ApplicationGroup status
func (in *BridgeTeam) GetStatusConditions() *[]metav1.Condition {
	return &in.Status.Conditions
}
