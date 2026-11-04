package fix

import (
	"slices"
	"strings"
	"testing"

	"github.com/kyverno/kyverno/cmd/cli/kubectl-kyverno/apis/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestFixTest_GlobalContextEntries_hasData(t *testing.T) {
	tests := []struct {
		name        string
		raw         []byte
		wantWarning bool // whether "has no data source" warning is expected
	}{
		{
			name:        "valid JSON object — no warning",
			raw:         []byte(`{"key":"value"}`),
			wantWarning: false,
		},
		{
			name:        "null literal — warning (treated as missing)",
			raw:         []byte(`null`),
			wantWarning: true,
		},
		{
			name:        "empty bytes — warning",
			raw:         []byte{},
			wantWarning: true,
		},
		{
			name:        "whitespace only — warning (must match ValidateGlobalContextEntries)",
			raw:         []byte("   \t\n"),
			wantWarning: true,
		},
		{
			name:        "whitespace around null — warning",
			raw:         []byte("  null  "),
			wantWarning: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var dataField *runtime.RawExtension
			if tc.raw != nil {
				dataField = &runtime.RawExtension{Raw: tc.raw}
			}

			test := v1alpha1.Test{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "cli.kyverno.io/v1alpha1",
					Kind:       "Test",
				},
				GlobalContextEntries: []v1alpha1.GlobalContextEntryValue{
					{Name: "g", Data: dataField},
				},
			}

			_, messages, err := FixTest(test, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			warnFound := false
			for _, m := range messages {
				if strings.Contains(m, "has no data source") {
					warnFound = true
					break
				}
			}

			if tc.wantWarning && !warnFound {
				t.Errorf("expected 'has no data source' warning but got none; messages: %v", messages)
			}
			if !tc.wantWarning && warnFound {
				t.Errorf("unexpected 'has no data source' warning; messages: %v", messages)
			}
		})
	}
}

func TestFixTest_SortsResourceLists(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		results []v1alpha1.TestResult
	}{
		{
			name: "single result",
			results: []v1alpha1.TestResult{{
				TestResultBase: v1alpha1.TestResultBase{Policy: "p", Rule: "r"},
				TestResultData: v1alpha1.TestResultData{
					Resources:          []string{"b", "a"},
					GeneratedResources: []string{"b-gen", "a-gen"},
				},
			}},
		},
		{
			name: "results with different policies",
			results: []v1alpha1.TestResult{
				{
					TestResultBase: v1alpha1.TestResultBase{Policy: "p2", Rule: "r"},
					TestResultData: v1alpha1.TestResultData{
						Resources:          []string{"b", "a"},
						GeneratedResources: []string{"b-gen", "a-gen"},
					},
				},
				{
					TestResultBase: v1alpha1.TestResultBase{Policy: "p1", Rule: "r"},
					TestResultData: v1alpha1.TestResultData{
						Resources:          []string{"d", "c"},
						GeneratedResources: []string{"d-gen", "c-gen"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, _, err := FixTest(v1alpha1.Test{Results: tt.results}, false)
			if err != nil {
				t.Fatal(err)
			}
			for _, r := range got.Results {
				if !slices.IsSorted(r.Resources) {
					t.Errorf("resources not sorted: %v", r.Resources)
				}
				if !slices.IsSorted(r.GeneratedResources) {
					t.Errorf("generatedResources not sorted: %v", r.GeneratedResources)
				}
			}
		})
	}
}
