package admissionpolicy

import (
	"testing"

	"github.com/kyverno/kyverno/pkg/clients/dclient"
	engineapi "github.com/kyverno/kyverno/pkg/engine/api"
	"github.com/stretchr/testify/require"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func emptyParamsTestClient(t *testing.T) dclient.Interface {
	t.Helper()
	// only a ConfigMap that does not match the binding's selector exists
	cm := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]any{
			"name":      "other",
			"namespace": "default",
			"labels":    map[string]any{"foo": "bar"},
		},
	}}
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{{Version: "v1", Resource: "configmaps"}: "ConfigMapList"}, cm)
	kube := kubefake.NewSimpleClientset(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}})
	return dclient.NewFakeClientWithDisco(dyn, kube, dclient.NewFakeDiscoveryClient(nil))
}

func emptyParamsTestPod() unstructured.Unstructured {
	return unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata":   map[string]any{"name": "probe", "namespace": "default"},
		"spec": map[string]any{
			"containers": []any{map[string]any{"name": "c", "image": "pause"}},
		},
	}}
}

func Test_ValidateWithClientEmptyParamSelection(t *testing.T) {
	pod := emptyParamsTestPod()
	gvk := schema.GroupVersionKind{Version: "v1", Kind: "Pod"}
	gvr := schema.GroupVersionResource{Version: "v1", Resource: "pods"}
	policy := &admissionregistrationv1.ValidatingAdmissionPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "empty-parameter-selection"},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicySpec{
			ParamKind: &admissionregistrationv1.ParamKind{APIVersion: "v1", Kind: "ConfigMap"},
			MatchConstraints: &admissionregistrationv1.MatchResources{
				ResourceRules: []admissionregistrationv1.NamedRuleWithOperations{{
					RuleWithOperations: admissionregistrationv1.RuleWithOperations{
						Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
						},
					},
				}},
			},
			Validations: []admissionregistrationv1.Validation{{Expression: "params.data.mode == 'enabled'"}},
		},
	}
	action := admissionregistrationv1.AllowAction
	binding := admissionregistrationv1.ValidatingAdmissionPolicyBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "binding"},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicyBindingSpec{
			PolicyName:        policy.Name,
			ValidationActions: []admissionregistrationv1.ValidationAction{admissionregistrationv1.Audit},
			ParamRef: &admissionregistrationv1.ParamRef{
				Selector:                &metav1.LabelSelector{MatchLabels: map[string]string{"policy-parameter": "unmatched"}},
				ParameterNotFoundAction: &action,
			},
		},
	}
	data := engineapi.NewValidatingAdmissionPolicyData(policy, binding)

	require.NotPanics(t, func() {
		_, err := Validate(data, pod, gvk, gvr, nil, emptyParamsTestClient(t), nil, false)
		require.NoError(t, err)
	})
}

func Test_MutateWithClientEmptyParamSelection(t *testing.T) {
	pod := emptyParamsTestPod()
	gvk := schema.GroupVersionKind{Version: "v1", Kind: "Pod"}
	gvr := schema.GroupVersionResource{Version: "v1", Resource: "pods"}
	policy := &admissionregistrationv1beta1.MutatingAdmissionPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "empty-parameter-selection"},
		Spec: admissionregistrationv1beta1.MutatingAdmissionPolicySpec{
			ParamKind: &admissionregistrationv1beta1.ParamKind{APIVersion: "v1", Kind: "ConfigMap"},
			MatchConstraints: &admissionregistrationv1beta1.MatchResources{
				ResourceRules: []admissionregistrationv1beta1.NamedRuleWithOperations{{
					RuleWithOperations: admissionregistrationv1beta1.RuleWithOperations{
						Operations: []admissionregistrationv1beta1.OperationType{admissionregistrationv1beta1.Create},
						Rule: admissionregistrationv1beta1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
						},
					},
				}},
			},
			ReinvocationPolicy: admissionregistrationv1beta1.NeverReinvocationPolicy,
			Mutations: []admissionregistrationv1beta1.Mutation{{
				PatchType: admissionregistrationv1beta1.PatchTypeApplyConfiguration,
				ApplyConfiguration: &admissionregistrationv1beta1.ApplyConfiguration{
					Expression: `Object{metadata: Object.metadata{labels: params.data}}`,
				},
			}},
		},
	}
	action := admissionregistrationv1beta1.AllowAction
	binding := admissionregistrationv1beta1.MutatingAdmissionPolicyBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "binding"},
		Spec: admissionregistrationv1beta1.MutatingAdmissionPolicyBindingSpec{
			PolicyName: policy.Name,
			ParamRef: &admissionregistrationv1beta1.ParamRef{
				Selector:                &metav1.LabelSelector{MatchLabels: map[string]string{"policy-parameter": "unmatched"}},
				ParameterNotFoundAction: &action,
			},
		},
	}
	data := engineapi.NewMutatingAdmissionPolicyData(policy, binding)

	require.NotPanics(t, func() {
		_, err := Mutate(data, pod, gvk, gvr, nil, emptyParamsTestClient(t), nil, false, false)
		require.NoError(t, err)
	})
}
