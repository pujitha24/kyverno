package engine

import (
	"testing"

	policieskyvernoio "github.com/kyverno/api/api/policies.kyverno.io"
	policiesv1beta1 "github.com/kyverno/api/api/policies.kyverno.io/v1beta1"
	"github.com/stretchr/testify/assert"
)

func TestPolicyExceptionRequestKey(t *testing.T) {
	t.Run("cluster-scoped ValidatingPolicy reference gets no namespace", func(t *testing.T) {
		ref := policiesv1beta1.PolicyRef{Name: "cpol", Kind: policieskyvernoio.ValidatingPolicyKind}
		key := policyExceptionRequestKey(ref, "test-ns")
		assert.Equal(t, "cpol", key.Name)
		assert.Equal(t, "", key.Namespace)
	})

	t.Run("NamespacedValidatingPolicy reference is scoped to the exception's namespace", func(t *testing.T) {
		ref := policiesv1beta1.PolicyRef{Name: "npol", Kind: policieskyvernoio.NamespacedValidatingPolicyKind}
		key := policyExceptionRequestKey(ref, "test-ns")
		assert.Equal(t, "npol", key.Name)
		assert.Equal(t, "test-ns", key.Namespace)
	})
}
