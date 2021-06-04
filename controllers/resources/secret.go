package resources

import (
	"context"
	"fmt"
	dbaasredhatcomv1alpha1 "github.com/CrunchyData/crunchy-bridge-operator/apis/dbaas.redhat.com/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	publicAPIKey     = "publicApiKey"
	privateAPISecret = "privateApiSecret"
)

// ConnectionAPIKeys encapsulates crunchybridge connectivity information that is necessary to generating token for performing API requests
type ConnectionAPIKeys struct {
	PublicKey     string
	PrivateSecret string
}

//ReadAPIKeysFromSecret
func ReadAPIKeysFromSecret(cient client.Client, ctx context.Context, inventory *dbaasredhatcomv1alpha1.CrunchyBridgeInventory) (ConnectionAPIKeys, error) {
	secret := &corev1.Secret{}
	selector := client.ObjectKey{
		Namespace: inventory.Spec.CredentialsRef.Namespace,
		Name:      inventory.Spec.CredentialsRef.Name,
	}

	if err := cient.Get(ctx, selector, secret); err != nil {
		return ConnectionAPIKeys{}, err
	}
	secretData := make(map[string]string)
	for k, v := range secret.Data {
		secretData[k] = string(v)
	}

	if err := validateAPIKeysSecret(selector, secretData); err != nil {
		return ConnectionAPIKeys{}, err
	}

	return ConnectionAPIKeys{
		PublicKey:     secretData["publicApiKey"],
		PrivateSecret: secretData["privateApiSecret"],
	}, nil
}

//validateAPIKeysSecret
func validateAPIKeysSecret(secretRef client.ObjectKey, secretData map[string]string) error {
	var missingFields []string
	requiredKeys := []string{publicAPIKey, privateAPISecret}

	for _, key := range requiredKeys {
		if _, ok := secretData[key]; !ok {
			missingFields = append(missingFields, key)
		}
	}

	if len(missingFields) > 0 {
		return fmt.Errorf("the following fields are missing in the Secret %v: %v", secretRef, missingFields)
	}
	return nil
}
