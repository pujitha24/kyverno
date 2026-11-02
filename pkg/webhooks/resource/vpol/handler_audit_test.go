package vpol

import (
	"context"
	"sync"
	"testing"

	"github.com/go-logr/logr"
	"github.com/kyverno/kyverno/pkg/breaker"
	celengine "github.com/kyverno/kyverno/pkg/cel/engine"
	"github.com/kyverno/kyverno/pkg/client/clientset/versioned/fake"
	engineapi "github.com/kyverno/kyverno/pkg/engine/api"
	"github.com/kyverno/kyverno/pkg/event"
	reportutils "github.com/kyverno/kyverno/pkg/utils/report"
	"github.com/kyverno/kyverno/pkg/webhooks/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type recordingEvents struct {
	mu    sync.Mutex
	infos []event.Info
}

func (r *recordingEvents) Add(infos ...event.Info) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.infos = append(r.infos, infos...)
}

func TestAudit_BlockedOnlyWhenDenyPolicyRejects(t *testing.T) {
	t.Parallel()
	reportutils.NewReportingConfig([]string{"pass", "fail"}, "validate")
	breaker.SetReportsBreaker(breaker.NewBreaker("test", func(context.Context) bool { return false }))
	deny := []admissionregistrationv1.ValidationAction{admissionregistrationv1.Deny}
	passed := *engineapi.RulePass("validation", engineapi.Validation, "ok", nil)
	failed := *engineapi.RuleFail("validation", engineapi.Validation, "denied", nil)
	tests := []struct {
		name        string
		rule        engineapi.RuleResponse
		wantReports int
	}{
		{name: "passing deny policy is reported", rule: passed, wantReports: 1},
		{name: "failing deny policy is blocked and not reported", rule: failed, wantReports: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resource := &unstructured.Unstructured{}
			resource.SetAPIVersion("v1")
			resource.SetKind("Pod")
			resource.SetName("pod")
			resource.SetNamespace("default")
			resource.SetUID("uid")
			client := fake.NewSimpleClientset()
			h := &handler{kyvernoClient: client, admissionReports: true, eventGen: &recordingEvents{}}
			req := handlers.AdmissionRequest{AdmissionRequest: admissionv1.AdmissionRequest{
				UID:       "req",
				Operation: admissionv1.Create,
				Kind:      metav1.GroupVersionKind{Version: "v1", Kind: "Pod"},
				Namespace: "default",
			}}
			engineReq := celengine.EngineRequest{Request: req.AdmissionRequest}
			response := celengine.EngineResponse{
				Resource: resource,
				Policies: []celengine.ValidatingPolicyResponse{testPolicy("p", nil, deny, tt.rule)},
			}
			h.audit(context.Background(), logr.Discard(), req, engineReq, response)
			list, err := client.ReportsV1().EphemeralReports("default").List(context.Background(), metav1.ListOptions{})
			require.NoError(t, err)
			assert.Len(t, list.Items, tt.wantReports)
		})
	}
}
