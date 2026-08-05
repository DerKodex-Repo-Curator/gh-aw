//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuggieEngineContract(t *testing.T) {
	registry := NewEngineRegistry()
	engine, err := registry.GetEngine("auggie")
	require.NoError(t, err)

	auggie, ok := engine.(*BehaviorDefinedEngine)
	require.True(t, ok, "Auggie should use the declarative behavior engine")
	assert.True(t, auggie.IsExperimental())
	assert.Contains(t, auggie.GetRequiredSecretNames(&WorkflowData{}), "AUGMENT_SESSION_AUTH")
	assert.Contains(t, auggie.GetAgentManifestFiles(), "AGENTS.md")
	assert.Contains(t, auggie.GetAgentManifestPathPrefixes(), ".augment/")

	installSteps := auggie.GetInstallationSteps(&WorkflowData{
		EngineConfig: &EngineConfig{ID: "auggie"},
	})
	installText := joinActionSteps(installSteps)
	assert.Contains(t, installText, "node-version: '24'")
	assert.Contains(t, installText, "@augmentcode/auggie@0.34.0")
	assert.Contains(t, installText, "auggie --version")

	overrideSteps := auggie.GetInstallationSteps(&WorkflowData{
		EngineConfig: &EngineConfig{ID: "auggie", Version: "0.33.0"},
	})
	assert.Contains(t, joinActionSteps(overrideSteps), "@augmentcode/auggie@0.33.0")
}

func TestAuggieExecutionUsesPromptFileAndMCP(t *testing.T) {
	registry := NewEngineRegistry()
	engine, err := registry.GetEngine("auggie")
	require.NoError(t, err)

	workflowData := &WorkflowData{
		EngineConfig: &EngineConfig{ID: "auggie"},
		Tools:        map[string]any{"github": true},
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/auggie.log")
	stepText := joinActionSteps(steps)

	assert.Contains(t, stepText, "auggie_harness.cjs")
	assert.Contains(t, stepText, `"--instruction-file"`)
	assert.Contains(t, stepText, `"--mcp-config"`)
	assert.Contains(t, stepText, "AUGGIE_MCP_CONFIG: /tmp/gh-aw/auggie-mcp.json")
	assert.Contains(t, stepText, "AUGMENT_DISABLE_AUTO_UPDATE: 1")
	assert.Contains(t, stepText, "AUGMENT_SESSION_AUTH: ${{ secrets.AUGMENT_SESSION_AUTH }}")
	assert.Contains(t, stepText, "--remove-tool")
	assert.Contains(t, stepText, "web-fetch")
	assert.Contains(t, stepText, "did not return valid JSON")
	assert.NotContains(t, stepText, "--exclude-env AUGMENT_SESSION_AUTH")
	assert.NotContains(t, stepText, `"$(cat /tmp/gh-aw/aw-prompts/prompt.txt)"`)
}

func TestAuggieSecretValidationOnlyRequiresSession(t *testing.T) {
	registry := NewEngineRegistry()
	engine, err := registry.GetEngine("auggie")
	require.NoError(t, err)

	step := engine.GetSecretValidationStep(&WorkflowData{
		Tools: map[string]any{"github": true},
	})
	stepText := strings.Join(step, "\n")
	assert.Contains(t, stepText, "AUGMENT_SESSION_AUTH")
	assert.NotContains(t, stepText, "GITHUB_MCP_SERVER_TOKEN")
}

func TestAuggieObjectFormCompilesWithVersionOverride(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-auggie-engine-*")
	workflowsDir := filepath.Join(tmpDir, constants.GetWorkflowDir())
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	workflow := `---
name: Test Auggie Engine
on:
  workflow_dispatch:
permissions:
  contents: read
engine:
  id: auggie
  version: "0.33.0"
tools:
  github:
    toolsets: [context]
---

# Task

Inspect the repository and report a noop result.
`
	workflowPath := filepath.Join(workflowsDir, "test-auggie.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(workflow), 0644))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowPath))

	lockContent, err := os.ReadFile(filepath.Join(workflowsDir, "test-auggie.lock.yml"))
	require.NoError(t, err)
	lockText := string(lockContent)
	assert.Contains(t, lockText, `GH_AW_INFO_ENGINE_ID: "auggie"`)
	assert.Contains(t, lockText, "@augmentcode/auggie@0.33.0")
	assert.Contains(t, lockText, "AUGMENT_SESSION_AUTH: ${{ secrets.AUGMENT_SESSION_AUTH }}")
	assert.Contains(t, lockText, "AUGGIE_MCP_CONFIG: /tmp/gh-aw/auggie-mcp.json")
	assert.Contains(t, lockText, "auggie_harness.cjs")
	assert.NotContains(t, lockText, "--exclude-env AUGMENT_SESSION_AUTH")
}

func joinActionSteps(steps []GitHubActionStep) string {
	var parts []string
	for _, step := range steps {
		parts = append(parts, strings.Join(step, "\n"))
	}
	return strings.Join(parts, "\n")
}
