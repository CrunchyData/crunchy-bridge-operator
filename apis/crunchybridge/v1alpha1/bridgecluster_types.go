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

// BridgeClusterSpec defines the desired state of BridgeCluster
type BridgeClusterSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Name represents the cluster name within Crunchy Bridge, must be
	// unique per team
	Name string `json:"name"`
	// TeamID identifies the target team in which to create the cluster.
	// Optional: defaults to the personal team of the operator's Crunchy Bridge account
	TeamID string `json:"team_id"`
	// Plan identifies the Crunchy Bridge provioning plan (e.g. hobby-2, standard-8)
	Plan string `json:"plan"`
	// Storage identifies the size of PostgreSQL database volume in megabytes
	StorageMB int `json:"storage"`
	// Provider identifies the desired cloud infrastructure provider (e.g. aws, gcp, azure)
	Provider string `json:"provider"`
	// Region identifies the requested deployment region within the provider (e.g. us-east-1)
	Region string `json:"region"`
	// Selects the major version of PostgreSQL to deploy (e.g. 12, 13)
	PGMajorVer int `json:"pg_major_version"`
	// Flags whether to deploy the additional nodes to enable high availability
	HighAvail bool `json:"enable_ha"`
}

// BridgeClusterStatus defines the observed state of BridgeCluster
type BridgeClusterStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Phase represents the cluster creation phase:
	//   pending - creation not yet started
	//   creating - provisioning in progress
	//   ready - cluster provisioning complete
	Phase string `json:"phase"`
	// Updated is the last status update from the controller, does not correlate to cluster.updated_at
	// TODO: Can this be moved to time.Time?
	Updated string `json:"last_update"`
	// Cluster represents cluster detail from Crunchy Bridge
	Cluster ClusterStatus `json:"cluster"`
	// Connect provides non-user specific connection information
	Connect Connection `json:"connection"`
}

type ClusterStatus struct {
	// Who
	//
	// ID represents the Crunchy Bridge cluster identifier
	ID string `json:"id"`
	// Name represents the cluster name provided in the request
	Name string `json:"name"`
	// TeamID represents the ID of the team which owns the cluster
	TeamID string `json:"team_id"`
	// What
	//
	// CPU represents the plan-allocated CPUs for the cluster
	CPU int `json:"cpu"`
	// MemoryGB represents the plan-allocated memory in gigabytes
	MemoryGB int `json:"memory"`
	// StorageMB represents the database volume size in megabytes
	StorageMB int `json:"storage"`
	// PGMajorVer represents the PostgreSQL major version number
	PGMajorVer int `json:"major_version"`
	// HighAvail represents whether the cluster has high availability enabled
	HighAvail bool `json:"ha_enabled"`
	// When
	// Created represents the cluster creation time as known to Crunchy Bridge
	// TODO: Can this be moved to time.Time?
	Created string `json:"created_at"`
	// Updated represents the last change time internal to Crunchy Bridge
	// TODO: Can this be moved to time.Time?
	Updated string `json:"updated_at"`
	// Where
	// ProviderID represents the infrastructure provider for the cluster
	ProviderID string `json:"provider_id"`
	// RegionID represents the region location for the cluster
	RegionID string `json:"region_id"`
}

type Connection struct {
	// URI represents the database connection string without user information
	// (e.g.postgres://p.fepkwudi6.example.com:5432/postgres)
	// TODO: Can this move to url.URL?
	URI string `json:"connect_string"`
	// Parent DB role identifies the name of the database role from which
	// bound user accounts inherit their permissions. Applications should
	// use SET ROLE to ensure DDL executed can be shared among binding roles
	ParentDBRole string `json:"parent_db_role"`
	// DatabaseName names the initial database created with the cluster
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
