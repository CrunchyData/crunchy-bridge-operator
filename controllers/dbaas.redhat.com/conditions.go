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
	dbaasredhatcomv1alpha1 "github.com/CrunchyData/crunchy-bridge-operator/apis/dbaas.redhat.com/v1alpha1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SpecSynced          string = "SpecSynced"
	BackendError        string = "BackendError"
	AuthenticationError string = "AuthenticationError"
	SyncOK              string = "SyncOK"
	ReadyForBinding     string = "ReadyForBinding"
	Ready               string = "Ready"
	NotFound            string = "NotFound"
	SuccessMessage      string = "Successfully listed crunchy bridge Inventories"
	SuccessConnection   string = "Successfully retrieved the connection detail\n"
)

// ObjectWithStatusConditions is an interface that describes kubernetes resource
// type structs with Status Conditions
type ObjectWithStatusConditions interface {
	GetStatusConditions() *[]metav1.Condition
}

// SetSyncCondition sets the given condition with the given status,
// reason and message on a resource.
func setStatusCondition(obj ObjectWithStatusConditions, condition string, status metav1.ConditionStatus, reason, message string) {
	conditions := obj.GetStatusConditions()

	newCondition := metav1.Condition{
		Type:    condition,
		Status:  status,
		Reason:  reason,
		Message: message,
	}

	apimeta.SetStatusCondition(conditions, newCondition)
}

// GetCondition return the condition with the passed condition type from
// the status object. If the condition is not already present, return nil
func GetConnectonCondition(inv *dbaasredhatcomv1alpha1.CrunchyBridgeConnection, condType string) *metav1.Condition {
	for i := range inv.Status.Conditions {
		if inv.Status.Conditions[i].Type == condType {
			return &inv.Status.Conditions[i]
		}
	}
	return nil
}

// GetCondition return the condition with the passed condition type from
// the status object. If the condition is not already present, return nil
func GetInventoryCondition(inv *dbaasredhatcomv1alpha1.CrunchyBridgeInventory, condType string) *metav1.Condition {
	for i := range inv.Status.Conditions {
		if inv.Status.Conditions[i].Type == condType {
			return &inv.Status.Conditions[i]
		}
	}
	return nil
}
