package dbaasredhatcom

import (
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SpecSynced     string = "SpecSynced"
	BackendError   string = "BackendError"
	SyncOK         string = "SyncOK"
	SuccessMessage string = "Successfully listed crunchy bridge Inventories"
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
