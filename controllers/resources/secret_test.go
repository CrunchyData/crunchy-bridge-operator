package resources

import (
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

func Test_validateConnectionSecret(t *testing.T) {
	err := validateAPIKeysSecret(ObjectKey("testNs", "testSecret"), map[string]string{})
	assert.EqualError(t, err, "the following fields are missing in the Secret testNs/testSecret: [publicApiKey privateApiSecret]")

	err = validateAPIKeysSecret(ObjectKey("testNs", "testSecret"), map[string]string{"publicApiKey": "foo"})
	assert.EqualError(t, err, "the following fields are missing in the Secret testNs/testSecret: [privateApiSecret]")

	err = validateAPIKeysSecret(ObjectKey("testNs", "testSecret"), map[string]string{"privateApiSecret": "foo"})
	assert.EqualError(t, err, "the following fields are missing in the Secret testNs/testSecret: [publicApiKey]")

	assert.NoError(t, validateAPIKeysSecret(ObjectKey("testNs", "testSecret"), map[string]string{"publicApiKey": "foo", "privateApiSecret": "foo:wq" +
		""}))

}
func ObjectKey(namespace, name string) client.ObjectKey {
	return types.NamespacedName{Name: name, Namespace: namespace}
}
