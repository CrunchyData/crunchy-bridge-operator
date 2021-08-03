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
package kubeadapter

import (
	"context"
	"fmt"

	"github.com/CrunchyData/crunchy-bridge-operator/internal/bridgeapi"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// kubeSecretCredentialProvider provides a LoginCred reflecting the Client's
// current knowledge of the secret
type KubeSecretCredentialProvider struct {
	// Client is a reference to the kube api client
	Client client.Client
	// Namespace in which the secret lives
	Namespace string
	// Object name of the secret
	Name string
	// KeyField identifies the secret's subkey to map to LoginCred.Key
	KeyField string
	// SecretField identifies the secret's subkey to map to LoginCred.Secret
	SecretField string
}

// TODO: untested
func (ks *KubeSecretCredentialProvider) ProvideCredential() bridgeapi.LoginCred {
	formedCred := bridgeapi.LoginCred{}
	secret := &corev1.Secret{}
	selector := client.ObjectKey{
		Namespace: ks.Namespace,
		Name:      ks.Name,
	}

	if err := ks.Client.Get(context.Background(), selector, secret); err != nil {
		formedCred.Error = fmt.Errorf("Error while getting the secret: %w", err)
		return formedCred
	}

	for k, v := range secret.Data {
		if k == ks.KeyField {
			formedCred.Key = string(v)
		}
		if k == ks.SecretField {
			formedCred.Secret = string(v)
		}
	}

	return formedCred
}
