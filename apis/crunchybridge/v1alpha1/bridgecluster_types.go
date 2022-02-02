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

const (
	PhaseUnknown  = ""
	PhasePending  = "Pending"
	PhaseCreating = "Creating"
	PhaseReady    = "Ready"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// defines the desired state of BridgeCluster
type BridgeClusterSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// represents the cluster name within Crunchy Bridge, must be unique per team
	// +kubebuilder:validation:MinLength=5
	Name string `json:"name"`
	// identifies the target team in which to create the cluster.
	// Defaults to the personal team of the operator's Crunchy Bridge account
	// +optional
	TeamID string `json:"team_id"`
	// identifies the Crunchy Bridge provioning plan (e.g. hobby-2, standard-8)
	Plan string `json:"plan"`
	// identifies the size of PostgreSQL database volume in gigabytes
	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:validation:Maximum=65535
	StorageGB int `json:"storage"`
	// identifies the desired cloud infrastructure provider
	// +kubebuilder:validation:Enum=aws;gcp;azure
	Provider string `json:"provider"`
	// identifies the requested deployment region within the provider (e.g. us-east-1)
	Region string `json:"region"`
	// selects the major version of PostgreSQL to deploy (e.g. 12, 13)
	// +kubebuilder:validation:Minimum=12
	PGMajorVer int `json:"pg_major_version"`
	// flags whether to deploy the additional nodes to enable high availability
	// +optional
	HighAvail bool `json:"enable_ha"`
}

// defines the observed state of BridgeCluster
type BridgeClusterStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	// represents the cluster creation phase:
	//     pending - creation not yet started
	//     creating - provisioning in progress
	//     ready - cluster provisioning complete
	Phase string `json:"phase"`
	// last status update from the controller, does not correlate to cluster.updated_at
	Updated string `json:"last_update"`
	// represents cluster detail from Crunchy Bridge
	Cluster ClusterStatus `json:"cluster"`
	// provides non-user specific connection information
	Connect Connection `json:"connection"`
}

type ClusterStatus struct {
	// represents the Crunchy Bridge cluster identifier
	ID string `json:"id"`
	// represents the cluster name provided in the request
	Name string `json:"name"`
	// represents the ID of the team which owns the cluster
	TeamID string `json:"team_id"`

	// represents the plan-allocated CPUs for the cluster
	CPU int `json:"cpu"`
	// represents the plan-allocated memory in gigabytes
	MemoryGB int `json:"memory"`
	// represents the database volume size in gigabytes
	StorageGB int `json:"storage"`
	// represents the PostgreSQL major version number
	PGMajorVer int `json:"major_version"`
	// represents whether the cluster has high availability enabled
	HighAvail bool `json:"ha_enabled"`

	// represents the cluster creation time as known to Crunchy Bridge
	Created string `json:"created_at"`
	// represents the last change time internal to Crunchy Bridge
	Updated string `json:"updated_at"`

	// represents the infrastructure provider for the cluster
	ProviderID string `json:"provider_id"`
	// represents the region location for the cluster
	RegionID string `json:"region_id"`
}

type Connection struct {
	// represents the database connection string without user information
	// (e.g.postgres://p.fepkwudi6.example.com:5432/postgres)
	URI string `json:"connect_string"`
	// identifies the name of the database role from which bound user accounts
	// inherit their permissions.
	// Applications should use SET ROLE to ensure DDL executed can be shared
	// among binding roles
	ParentDBRole string `json:"parent_db_role"`
	// identifies the initial database created with the cluster
	DatabaseName string `json:"database_name"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BridgeCluster is the Schema for the bridgeclusters API
type BridgeCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BridgeClusterSpec   `json:"spec,omitempty"`
	Status BridgeClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BridgeClusterList contains a list of BridgeCluster
type BridgeClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BridgeCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BridgeCluster{}, &BridgeClusterList{})
}
