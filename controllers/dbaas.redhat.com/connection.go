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
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ptr "k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dbaasredhatcomv1alpha1 "github.com/CrunchyData/crunchy-bridge-operator/apis/dbaas.redhat.com/v1alpha1"
	"github.com/CrunchyData/crunchy-bridge-operator/internal/bridgeapi"
	dbaasv1alpha1 "github.com/RHEcosystemAppEng/dbaas-operator/api/v1alpha1"
)

const (
	HOSTKEYNAME         string = "host"
	PORTKEYNAME         string = "port"
	DBKEYNAME           string = "database"
	TYPEKEYNAME         string = "type"
	USERNAMEKEYNAME     string = "username"
	PASSWORDKEYNAME     string = "password"
	DATABASESERVICETYPE string = "postgresql"
	PROVIDERVALUE              = "rhoda/crunchy bridge"
	PROVIDERKEY                = "provider"
)

// connectionDetails
func (r *CrunchyBridgeConnectionReconciler) connectionDetails(instanceID string, connection *dbaasredhatcomv1alpha1.CrunchyBridgeConnection, bridgeapi *bridgeapi.Client, req ctrl.Request, logger logr.Logger) error {

	if r.isBindingExist(connection) && connection.Status.Binding != nil {
		return nil
	}

	connectionRole, err := bridgeapi.DefaultConnRole(instanceID)

	if err != nil {
		logger.Error(err, "Error in getting the connectionRole")
		return err
	}

	if connection.Status.Binding == nil {
		secret := getOwnedSecret(connection, connectionRole.URI, connectionRole.Name, connectionRole.Password)
		err := r.Client.Create(context.Background(), secret, &client.CreateOptions{})
		if err != nil {
			logger.Error(err, "Error in creating the secret")
			return err
		}
		connection.Status.Binding = &corev1.LocalObjectReference{Name: secret.Name}
	}
	return nil

}

// getOwnedSecret returns a secret object for database credentials with ownership set
func getOwnedSecret(connection *dbaasredhatcomv1alpha1.CrunchyBridgeConnection, connectionString, username, password string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind: "Opaque",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "crunchy-bridge-db-credentials-",
			Namespace:    connection.Namespace,
			Labels: map[string]string{
				"managed-by":               "crunchy-bridge-operator",
				"owner":                    connection.Name,
				"owner.kind":               connection.Kind,
				"owner.namespace":          connection.Namespace,
				dbaasv1alpha1.TypeLabelKey: dbaasv1alpha1.TypeLabelValue,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					UID:                connection.GetUID(),
					APIVersion:         connection.APIVersion,
					BlockOwnerDeletion: ptr.BoolPtr(false),
					Controller:         ptr.BoolPtr(true),
					Kind:               connection.Kind,
					Name:               connection.Name,
				},
			},
		},
		Type: corev1.SecretType(fmt.Sprintf("servicebinding.io/%s", DATABASESERVICETYPE)),
		Data: connectionSecretData(connectionString, username, password),
	}
}

// connectionSecretData
func connectionSecretData(connectionString, username, password string) map[string][]byte {
	bindingParamsMap := make(map[string][]byte)
	u, err := url.Parse(connectionString)
	if err != nil {
		return bindingParamsMap
	}
	host, port, _ := net.SplitHostPort(u.Host)
	bindingParamsMap[TYPEKEYNAME] = []byte(DATABASESERVICETYPE)
	bindingParamsMap[PROVIDERKEY] = []byte(PROVIDERVALUE)
	bindingParamsMap[HOSTKEYNAME] = []byte(host)
	bindingParamsMap[PORTKEYNAME] = []byte(port)
	bindingParamsMap[DBKEYNAME] = []byte(strings.TrimLeft(u.Path, "/"))
	bindingParamsMap[USERNAMEKEYNAME] = []byte(username)
	bindingParamsMap[PASSWORDKEYNAME] = []byte(password)
	return bindingParamsMap

}

// isBindingExist checking if binding already exits
func (r *CrunchyBridgeConnectionReconciler) isBindingExist(connection *dbaasredhatcomv1alpha1.CrunchyBridgeConnection) bool {

	cond := GetConnectonCondition(connection, string(ReadyForBinding))
	if cond != nil && cond.Status == metav1.ConditionTrue {
		return true
	}

	return false
}
