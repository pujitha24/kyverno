package ivpol

import (
	"context"
	"errors"
	"testing"

	policiesv1beta1 "github.com/kyverno/api/api/policies.kyverno.io/v1beta1"
	celengine "github.com/kyverno/kyverno/pkg/cel/engine"
	engineapi "github.com/kyverno/kyverno/pkg/engine/api"
	eval "github.com/kyverno/kyverno/pkg/image/verification/evaluator"
	"github.com/kyverno/kyverno/pkg/toggle"
	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
)

type forceIgnoreToggles struct {
	toggle.Toggles
	force bool
}

func (f forceIgnoreToggles) ForceFailurePolicyIgnore() bool { return f.force }

func TestErrorResultHonorsFailurePolicy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		failurePolicy *admissionregistrationv1.FailurePolicyType
		forceIgnore   bool
		allowed       bool
	}{
		{name: "default is Fail", allowed: false},
		{name: "Fail", failurePolicy: ptr.To(admissionregistrationv1.Fail), allowed: false},
		{name: "Ignore", failurePolicy: ptr.To(admissionregistrationv1.Ignore), allowed: true},
		{name: "Fail with forceFailurePolicyIgnore", failurePolicy: ptr.To(admissionregistrationv1.Fail), forceIgnore: true, allowed: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			policy := &policiesv1beta1.ImageValidatingPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "p"},
				Spec:       policiesv1beta1.ImageValidatingPolicySpec{FailurePolicy: tt.failurePolicy},
			}
			ctx := toggle.NewContext(context.Background(), forceIgnoreToggles{Toggles: toggle.FromContext(context.Background()), force: tt.forceIgnore})
			request := celengine.EngineRequest{Request: admissionv1.AdmissionRequest{UID: "uid"}}
			response := eval.ImageVerifyEngineResponse{Policies: []eval.ImageVerifyPolicyResponse{{
				Policy:  policy,
				Actions: sets.New(admissionregistrationv1.Deny),
				Result:  *engineapi.RuleError("evaluation", engineapi.ImageVerify, "failed to evaluate policy", errors.New("registry unavailable"), nil),
			}}}
			h := &handler{}
			assert.Equal(t, tt.allowed, h.validationResponse(ctx, request, response).Allowed)
			assert.Equal(t, tt.allowed, h.mutationResponse(ctx, request, response, nil).Allowed)
		})
	}
}

func TestFailResultDeniesRegardlessOfFailurePolicy(t *testing.T) {
	t.Parallel()
	policy := &policiesv1beta1.ImageValidatingPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "p"},
		Spec:       policiesv1beta1.ImageValidatingPolicySpec{FailurePolicy: ptr.To(admissionregistrationv1.Ignore)},
	}
	request := celengine.EngineRequest{Request: admissionv1.AdmissionRequest{UID: "uid"}}
	response := eval.ImageVerifyEngineResponse{Policies: []eval.ImageVerifyPolicyResponse{{
		Policy:  policy,
		Actions: sets.New(admissionregistrationv1.Deny),
		Result:  *engineapi.RuleFail("evaluation", engineapi.ImageVerify, "unsigned", nil),
	}}}
	h := &handler{}
	assert.False(t, h.validationResponse(context.Background(), request, response).Allowed)
}
