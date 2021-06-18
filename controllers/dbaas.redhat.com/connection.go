package dbaasredhatcom

import (
	"context"
	dbaasredhatcomv1alpha1 "github.com/CrunchyData/crunchy-bridge-operator/apis/dbaas.redhat.com/v1alpha1"
	"github.com/CrunchyData/crunchy-bridge-operator/internal/bridgeapi"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ptr "k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	INSTANCE_ID string = "instanceId"
)

func (r *CrunchyBridgeConnectionReconciler) connectionDetails(instanceID string, connection *dbaasredhatcomv1alpha1.CrunchyBridgeConnection, bridgeapi *bridgeapi.Client, req ctrl.Request, logger logr.Logger) error {

	if r.isBindingExist(instanceID, connection, logger) {
		return nil
	}

	connectionRole, err := bridgeapi.DefaultConnRole(instanceID)

	if err != nil {
		logger.Error(err, "Error in getting the connectionRole")
		return err
	}
	secret := getOwnedSecret(connection, connectionRole.Name, connectionRole.Password)
	secretCreated, err := r.Clientset.CoreV1().Secrets(req.Namespace).Create(context.Background(), secret, metav1.CreateOptions{})
	if err != nil {
		logger.Error(err, "Error in creating the secret")
		return err
	}
	connection.Status.ConnectionString = connectionRole.URI
	connection.Status.CredentialsRef = &corev1.LocalObjectReference{Name: secretCreated.Name}
	connection.Status.ConnectionInfo = map[string]string{"instanceId": instanceID}

	return nil

}

// getOwnedSecret returns a secret object for database credentials with ownership set
func getOwnedSecret(connection *dbaasredhatcomv1alpha1.CrunchyBridgeConnection, username, password string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind: "Opaque",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "crunchy-bridge-db-user-",
			Namespace:    connection.Namespace,
			Labels: map[string]string{
				"managed-by":      "crunchy-bridge-operator",
				"owner":           connection.Name,
				"owner.kind":      connection.Kind,
				"owner.namespace": connection.Namespace,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					UID:                connection.GetUID(),
					APIVersion:         "dbaas.redhat.com/v1alpha1",
					BlockOwnerDeletion: ptr.BoolPtr(false),
					Controller:         ptr.BoolPtr(true),
					Kind:               connection.Kind,
					Name:               connection.Name,
				},
			},
		},
		Data: map[string][]byte{
			"username": []byte(username),
			"password": []byte(password),
		},
	}
}

func (r *CrunchyBridgeConnectionReconciler) isBindingExist(instanceID string, connection *dbaasredhatcomv1alpha1.CrunchyBridgeConnection, logger logr.Logger) bool {

	existingInsId, ok := connection.Status.ConnectionInfo[INSTANCE_ID]
	if !ok {
		return ok
	}
	cond := GetConnectonCondition(connection, string(ReadyForBinding))
	if existingInsId == instanceID && cond != nil && cond.Status == metav1.ConditionTrue {
		return true
	}

	// remove the previous created secret if instance id changed
	existingtSecret := connection.Status.CredentialsRef.Name
	found := &corev1.Secret{}
	err := r.Get(context.Background(), types.NamespacedName{Name: existingtSecret, Namespace: connection.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		return false
	}
	if err := r.Client.Delete(context.Background(), found); err != nil {
		logger.Error(err, "Failed to delete secret", "secretName", existingtSecret)
		return false
	}
	return false
}
