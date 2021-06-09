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
package bridgeapi

import (
	"errors"
	"time"
)

var (
	ErrorBadRequest = errors.New("Invalid request")
	ErrorConflict   = errors.New("Non-unique name specified in request")
	ErrorAPIUnset   = errors.New("No API target URL set")
)

var (
	routeClusters string = "/clusters"
)

type ClusterState string

const (
	StateUnknown  ClusterState = "unknown"
	StateCreating ClusterState = "creating"
	StateReady    ClusterState = "ready"
)

type CreateRequest struct {
	Name             string `json:"name"`
	TeamID           string `json:"team_id"`
	Plan             string `json:"plan_id"`
	StorageMB        int    `json:"storage"`
	Provider         string `json:"provider_id"`
	Region           string `json:"region_id"`
	PGMajorVersion   int    `json:"major_version"`
	HighAvailability bool   `json:"is_ha"`
}

type ClusterList struct {
	Clusters []ClusterDetail `json:"clusters"`
	Count    int             `json:"total_count"`
}

type ClusterDetail struct {
	CPU              int              `json:"cpu"`
	Created          time.Time        `json:"created_at"`
	ID               string           `json:"id"`
	HighAvailability bool             `json:"is_ha"`
	PGMajorVersion   int              `json:"major_version"`
	MemoryGB         int              `json:"memory"`
	Name             string           `json:"name"`
	OldestBackup     time.Time        `json:"oldest_backup"`
	ProviderID       string           `json:"provider_id"`
	RegionID         string           `json:"region_id"`
	State            string           `json:"state"` // Leave as string until graceful error handling
	StorageMB        int              `json:"storage"`
	TeamID           string           `json:"team_id"`
	Updated          time.Time        `json:"updated_at"`
	Instances        []InstanceDetail `json:"instances"` // From single-cluster detail
	Replicas         []ClusterDetail  `json:"replicas"`  // From cluster listing
}

type InstanceDetail struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ProviderID string `json:"provider_id"`
	RegionID   string `json:"region_id"`
	Type       string `json:"type"` // primary, read_replica
	URL        string `json:"url"`
}
