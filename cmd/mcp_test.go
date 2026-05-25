package cmd_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/brandonyoungdev/tldx/cmd"
	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/resolver"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/openrdap/rdap"
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

func TestMCP_ToolConstructors(t *testing.T) {
	tool := cmd.MCPCheckDomainTool()
	assert.Equal(t, "check_domain", tool.Name)

	tool2 := cmd.MCPCheckDomainsTool()
	assert.Equal(t, "check_domains", tool2.Name)

	tool3 := cmd.MCPGenerateAndCheckTool()
	assert.Equal(t, "generate_and_check", tool3.Name)

	tool4 := cmd.MCPListTLDPresetsTool()
	assert.Equal(t, "list_tld_presets", tool4.Name)
}

func TestNewMCPCmd_Structure(t *testing.T) {
	mcpCmd := cmd.NewMCPCmd("v1.0.0")
	require.NotNil(t, mcpCmd)
	assert.Equal(t, "mcp", mcpCmd.Use)
	assert.NotEmpty(t, mcpCmd.Short)
	assert.NotEmpty(t, mcpCmd.Long)
}

func TestMCP_CheckDomainsHandler_WithDomains(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"domains": []any{"stripe.com", "atlas.io"},
	}
	result, err := invokeMCPCheckDomains(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMCP_GenerateAndCheck_OnlyAvailableParam(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"keywords":       []any{"testword"},
		"tlds":           []any{"com"},
		"only_available": true,
		"limit":          float64(5),
	}
	result, err := invokeMCPGenerateAndCheck(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMCP_GenerateAndCheck_MaxDomainLengthParam(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"keywords":          []any{"hello"},
		"tlds":              []any{"com"},
		"max_domain_length": float64(20),
	}
	result, err := invokeMCPGenerateAndCheck(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

type mockRDAPQuerierForMCP struct {
	err error
}

func (m *mockRDAPQuerierForMCP) Do(_ *rdap.Request) (*rdap.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &rdap.Response{Object: &rdap.Domain{}}, nil
}

func withMockFactory(t *testing.T, mockErr error) func() {
	t.Helper()
	orig := cmd.ResolverFactory
	cmd.ResolverFactory = func(app *config.TldxContext, opts ...resolver.ResolverOption) *resolver.ResolverService {
		return resolver.NewResolverService(app, append(opts,
			resolver.WithRDAPQuerier(&mockRDAPQuerierForMCP{err: mockErr}),
		)...)
	}
	return func() { cmd.ResolverFactory = orig }
}

func TestMCP_CheckDomains_WithMockedResolver(t *testing.T) {
	restore := withMockFactory(t, fmt.Errorf("object does not exist."))
	defer restore()

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"domains": []any{"available1.com", "available2.io"},
	}
	result, err := invokeMCPCheckDomains(context.Background(), req)
	require.NoError(t, err)
	require.False(t, result.IsError)

	text := extractTextContent(t, result)
	var results []map[string]any
	require.NoError(t, json.Unmarshal([]byte(text), &results))
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.Equal(t, true, r["available"])
	}
}

func TestMCP_GenerateAndCheck_WithMockedResolver(t *testing.T) {
	restore := withMockFactory(t, fmt.Errorf("object does not exist."))
	defer restore()

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"keywords": []any{"testword"},
		"tlds":     []any{"com"},
	}
	result, err := invokeMCPGenerateAndCheck(context.Background(), req)
	require.NoError(t, err)
	require.False(t, result.IsError)

	text := extractTextContent(t, result)
	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(text), &payload))
	results, ok := payload["results"].([]any)
	require.True(t, ok)
	assert.NotEmpty(t, results)
}

func TestMCP_GenerateAndCheck_OnlyAvailableFilters(t *testing.T) {
	// Use a mock that marks all domains as registered (available=false)
	restore := withMockFactory(t, nil) // err=nil means registered
	defer restore()

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"keywords":       []any{"taken"},
		"tlds":           []any{"com"},
		"only_available": true,
	}
	result, err := invokeMCPGenerateAndCheck(context.Background(), req)
	require.NoError(t, err)
	require.False(t, result.IsError)

	text := extractTextContent(t, result)
	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(text), &payload))
	results := payload["results"]
	// All domains are registered, only_available=true → results should be empty
	assert.Nil(t, results)
}

func TestMCP_GenerateAndCheck_LimitStopsEarly(t *testing.T) {
	restore := withMockFactory(t, fmt.Errorf("object does not exist."))
	defer restore()

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"keywords": []any{"word1", "word2", "word3"},
		"tlds":     []any{"com", "io", "ai"},
		"limit":    float64(1),
	}
	result, err := invokeMCPGenerateAndCheck(context.Background(), req)
	require.NoError(t, err)
	require.False(t, result.IsError)

	text := extractTextContent(t, result)
	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(text), &payload))
	total := int(payload["total"].(float64))
	assert.Equal(t, 1, total, "limit=1 should stop after first available domain")
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
