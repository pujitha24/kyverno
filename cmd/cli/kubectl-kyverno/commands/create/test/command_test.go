package test

import (
	"bytes"
	"io"
	"path/filepath"
	"strings"
	"testing"

	clitest "github.com/kyverno/kyverno/cmd/cli/kubectl-kyverno/test"
	"github.com/stretchr/testify/assert"
)

func TestCommand(t *testing.T) {
	cmd := Command()
	assert.NotNil(t, cmd)
	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestCommandWithArgs(t *testing.T) {
	cmd := Command()
	assert.NotNil(t, cmd)
	cmd.SetArgs([]string{"foo"})
	err := cmd.Execute()
	assert.Error(t, err)
}

func TestCommandWithInvalidArg(t *testing.T) {
	cmd := Command()
	assert.NotNil(t, cmd)
	b := bytes.NewBufferString("")
	cmd.SetErr(b)
	cmd.SetArgs([]string{"foo"})
	err := cmd.Execute()
	assert.Error(t, err)
	out, err := io.ReadAll(b)
	assert.NoError(t, err)
	expected := `Error: unknown command "foo" for "test"`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(out)))
}

func TestCommandWithInvalidFlag(t *testing.T) {
	cmd := Command()
	assert.NotNil(t, cmd)
	b := bytes.NewBufferString("")
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--xxx"})
	err := cmd.Execute()
	assert.Error(t, err)
	out, err := io.ReadAll(b)
	assert.NoError(t, err)
	expected := `Error: unknown flag: --xxx`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(out)))
}

func TestCommandHelp(t *testing.T) {
	cmd := Command()
	assert.NotNil(t, cmd)
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	assert.NoError(t, err)
	out, err := io.ReadAll(b)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(out), cmd.Long))
}

func TestCommandOutput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		args      []string
		resources []string
		patched   string
	}{{
		name:      "namespaced",
		args:      []string{"--pass", "pol,rule,res,ns,Pod"},
		resources: []string{"ns/res"},
	}, {
		name:      "cluster scoped",
		args:      []string{"--fail", "pol,rule,res,,Namespace"},
		resources: []string{"res"},
	}, {
		name:      "patched",
		args:      []string{"--pass", "pol,rule,res,ns,Pod,patched.yaml"},
		resources: []string{"ns/res"},
		patched:   "patched.yaml",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "kyverno-test.yaml")
			cmd := Command()
			cmd.SetArgs(append([]string{"-p", "policy.yaml", "-r", "resource.yaml", "-o", path}, tt.args...))
			assert.NoError(t, cmd.Execute())
			cases := clitest.LoadTest(nil, path)
			if !assert.Len(t, cases, 1) || !assert.NoError(t, cases[0].Err) {
				return
			}
			got := cases[0].Test
			assert.Equal(t, "cli.kyverno.io/v1alpha1", got.APIVersion)
			assert.Equal(t, "Test", got.Kind)
			assert.Equal(t, "test-name", got.ObjectMeta.Name)
			if assert.Len(t, got.Results, 1) {
				assert.Equal(t, tt.resources, got.Results[0].Resources)
				assert.Equal(t, tt.patched, got.Results[0].PatchedResources)
			}
		})
	}
}
