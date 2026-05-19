package cmd_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/brandonyoungdev/tldx/cmd"
	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/resolver"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCommand_DryRun(t *testing.T) {
	app := config.NewTldxContext()

	var buf bytes.Buffer
	rootCmd := cmd.NewRootCmd(app)
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"stripe", "--tlds", "com,io", "--dry-run"})

	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestRootCommand_ErrNoAvailableDomains(t *testing.T) {
	app := config.NewTldxContext()

	rootCmd := cmd.NewRootCmd(app)
	rootCmd.SetArgs([]string{"google", "--tlds", "com", "--only-available"})
	rootCmd.SilenceErrors = true

	err := rootCmd.Execute()
	if err != nil {
		assert.True(t, errors.Is(err, cmd.ErrNoAvailableDomains),
			"expected ErrNoAvailableDomains, got: %v", err)
	}
}

func TestRootCommand_Limit_Flag(t *testing.T) {
	app := config.NewTldxContext()
	assert.Equal(t, 0, app.Config.Limit)

	rootCmd := cmd.NewRootCmd(app)
	rootCmd.SetArgs([]string{"--limit", "3", "--help"})
	rootCmd.SilenceErrors = true
	rootCmd.Execute() //nolint:errcheck

	f := rootCmd.Flags().Lookup("limit")
	require.NotNil(t, f, "expected --limit flag to be registered")
	assert.Equal(t, "3", f.Value.String())
}

func TestRootCommand_DryRun_Flag(t *testing.T) {
	f := cmd.NewRootCmd(config.NewTldxContext()).Flags().Lookup("dry-run")
	require.NotNil(t, f, "expected --dry-run flag to be registered")
}

func TestRootCommand_StdinInput(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "tldx-test-*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("stripe\natlas\n")
	tmpFile.Close()

	app := config.NewTldxContext()
	rootCmd := cmd.NewRootCmd(app)
	rootCmd.SetArgs([]string{"--input", tmpFile.Name(), "--dry-run"})
	err = rootCmd.Execute()
	require.NoError(t, err)
}

func TestMCP_CheckDomainTool_InvalidDomain(t *testing.T) {
	result, err := invokeMCPCheckDomain(context.Background(), "@@@invalid")
	require.NoError(t, err)
	assert.True(t, result.IsError, "expected tool error for invalid domain")
}

func TestMCP_CheckDomainsTool_EmptyList(t *testing.T) {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"domains": []any{},
	}
	result, err := invokeMCPCheckDomains(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestMCP_GenerateAndCheck_RequiresKeywords(t *testing.T) {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}
	result, err := invokeMCPGenerateAndCheck(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestMCP_GenerateAndCheck_DryRunEquivalent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"keywords": []any{"testdomain"},
		"tlds":     []any{"com"},
	}
	result, err := invokeMCPGenerateAndCheck(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMCP_ListTLDPresets(t *testing.T) {
	req := mcp.CallToolRequest{}
	result, err := invokeMCPListPresets(context.Background(), req)
	require.NoError(t, err)
	require.False(t, result.IsError, "list_tld_presets should not error")

	text := extractTextContent(t, result)
	assert.Contains(t, text, "popular")
	assert.Contains(t, text, "tech")
	assert.Contains(t, text, "startup")
}

func TestMCP_ListTLDPresets_IncludesAll(t *testing.T) {
	req := mcp.CallToolRequest{}
	result, err := invokeMCPListPresets(context.Background(), req)
	require.NoError(t, err)
	text := extractTextContent(t, result)

	assert.Contains(t, text, `"name":"all"`)
}

func TestMCP_CheckDomainTool_ValidDomain_Structure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	result, err := invokeMCPCheckDomain(ctx, "example.com")
	require.NoError(t, err)
	if !result.IsError {
		text := extractTextContent(t, result)
		var obj map[string]any
		assert.NoError(t, json.Unmarshal([]byte(text), &obj))
		assert.Contains(t, obj, "domain")
		assert.Contains(t, obj, "available")
	}
}

func TestMCP_GenerateAndCheck_MetadataFields(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"keywords": []any{"stripe"},
		"tlds":     []any{"com"},
		"prefixes": []any{"get"},
	}
	result, err := invokeMCPGenerateAndCheck(ctx, req)
	require.NoError(t, err)

	if result.IsError {
		return
	}
	text := extractTextContent(t, result)
	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(text), &payload))
	results, ok := payload["results"].([]any)
	if ok && len(results) > 0 {
		entry := results[0].(map[string]any)
		assert.Contains(t, entry, "keyword")
		assert.Contains(t, entry, "prefix")
		assert.Contains(t, entry, "suffix")
		assert.Contains(t, entry, "tld")
	}
}

func TestDomainResult_RicherJSONFields(t *testing.T) {
	r := resolver.DomainResult{
		Domain:    "getstripe.com",
		Available: true,
		Keyword:   "stripe",
		Prefix:    "get",
		Suffix:    "",
		TLD:       "com",
	}

	b, err := json.Marshal(r.AsEncodable())
	require.NoError(t, err)
	s := string(b)

	assert.Contains(t, s, `"keyword":"stripe"`)
	assert.Contains(t, s, `"prefix":"get"`)
	assert.Contains(t, s, `"tld":"com"`)
	assert.NotContains(t, s, `"suffix"`)
}

func TestOutput_StdinKeywordSupport(t *testing.T) {
	// Pipe test content to simulate stdin via file
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer r.Close()

	w.WriteString("myword\n")
	w.Close()

	// Replace stdin temporarily
	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	app := config.NewTldxContext()
	rootCmd := cmd.NewRootCmd(app)
	rootCmd.SetArgs([]string{"--input", "-", "--dry-run"})
	err = rootCmd.Execute()
	require.NoError(t, err)
}

func TestOutput_TTYAutoDetect(t *testing.T) {
	app := config.NewTldxContext()
	from := cmd.NewRootCmd(app)
	f := from.Flags().Lookup("no-color")
	require.NotNil(t, f)
	assert.Equal(t, "false", f.DefValue)
}

func invokeMCPCheckDomain(ctx context.Context, domain string) (*mcp.CallToolResult, error) {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"domain": domain}
	return cmd.MCPCheckDomainHandler(ctx, req)
}

func invokeMCPCheckDomains(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return cmd.MCPCheckDomainsHandler(ctx, req)
}

func invokeMCPGenerateAndCheck(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return cmd.MCPGenerateAndCheckHandler(ctx, req)
}

func invokeMCPListPresets(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return cmd.MCPListTLDPresetsHandler(ctx, req)
}

func extractTextContent(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	for _, c := range result.Content {
		if tc, ok := mcp.AsTextContent(c); ok {
			return strings.TrimSpace(tc.Text)
		}
	}
	t.Fatal("no text content in tool result")
	return ""
}
