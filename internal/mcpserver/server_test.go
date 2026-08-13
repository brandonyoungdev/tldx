package mcpserver_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/mcpserver"
	"github.com/brandonyoungdev/tldx/internal/resolver"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/openrdap/rdap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isolateConfig points TLDX_CONFIG at a scratch file so a test never touches
// the real user config.
func isolateConfig(t *testing.T, contents string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if contents != "" {
		require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	}
	t.Setenv("TLDX_CONFIG", path)
}

type mockRDAP struct {
	err   error
	calls atomic.Int64
}

func (m *mockRDAP) Do(_ *rdap.Request) (*rdap.Response, error) {
	m.calls.Add(1)
	if m.err != nil {
		return nil, m.err
	}
	return &rdap.Response{Object: &rdap.Domain{}}, nil
}

// What RDAP returns for an unregistered domain.
var errNotFound = fmt.Errorf("object does not exist.")

// newClient starts an in-process MCP client against a real server, so tests go
// through the tools as registered rather than calling handlers directly.
func newClient(t *testing.T, opts ...mcpserver.Option) *client.Client {
	t.Helper()

	c, err := client.NewInProcessClient(mcpserver.New("test", opts...))
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	require.NoError(t, c.Start(ctx))
	_, err = c.Initialize(ctx, mcp.InitializeRequest{})
	require.NoError(t, err)

	return c
}

func withRDAP(m *mockRDAP, extra ...resolver.ResolverOption) mcpserver.Option {
	return mcpserver.WithResolverFactory(
		func(app *config.TldxContext, opts ...resolver.ResolverOption) *resolver.ResolverService {
			opts = append(opts, resolver.WithRDAPQuerier(m))
			return resolver.NewResolverService(app, append(opts, extra...)...)
		})
}

func callTool(t *testing.T, c *client.Client, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args

	res, err := c.CallTool(ctx, req)
	require.NoError(t, err)
	return res
}

func textOf(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	for _, c := range res.Content {
		if tc, ok := mcp.AsTextContent(c); ok {
			return strings.TrimSpace(tc.Text)
		}
	}
	t.Fatal("no text content in tool result")
	return ""
}

func decode(t *testing.T, res *mcp.CallToolResult) mcpserver.CheckResponse {
	t.Helper()
	require.False(t, res.IsError, "unexpected tool error: %s", textOf(t, res))

	var out mcpserver.CheckResponse
	require.NoError(t, json.Unmarshal([]byte(textOf(t, res)), &out))
	return out
}

func listTools(t *testing.T, c *client.Client) []mcp.Tool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	require.NoError(t, err)
	return res.Tools
}

func toolNamed(t *testing.T, tools []mcp.Tool, name string) mcp.Tool {
	t.Helper()
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %q not registered", name)
	return mcp.Tool{}
}

func TestServer_ExposesExactlyTwoTools(t *testing.T) {
	isolateConfig(t, "")

	names := []string{}
	for _, tool := range listTools(t, newClient(t)) {
		names = append(names, tool.Name)
	}

	assert.ElementsMatch(t, []string{"check_domains", "generate_and_check"}, names)
}

func TestServer_ToolsAreAnnotatedReadOnly(t *testing.T) {
	isolateConfig(t, "")

	for _, tool := range listTools(t, newClient(t)) {
		require.NotNil(t, tool.Annotations.ReadOnlyHint, "%s: missing readOnlyHint", tool.Name)
		assert.True(t, *tool.Annotations.ReadOnlyHint, "%s should be read-only", tool.Name)

		require.NotNil(t, tool.Annotations.OpenWorldHint, "%s: missing openWorldHint", tool.Name)
		assert.True(t, *tool.Annotations.OpenWorldHint, "%s reaches the network", tool.Name)

		assert.NotEmpty(t, tool.Annotations.Title, "%s: missing title", tool.Name)
	}
}

// mcp-go silently drops schema fields it cannot describe by reflection, which
// with "additionalProperties": false makes strict clients reject responses.
func TestServer_OutputSchemaDescribesResults(t *testing.T) {
	isolateConfig(t, "")

	for _, tool := range listTools(t, newClient(t)) {
		schema := tool.OutputSchema

		require.Contains(t, schema.Properties, "results", "%s: results dropped from output schema", tool.Name)
		assert.Contains(t, schema.Required, "results", "%s: results should be required", tool.Name)

		for _, key := range []string{"checked", "available_count", "taken_count", "truncated", "dry_run", "note"} {
			assert.Contains(t, schema.Properties, key, "%s: %s missing from output schema", tool.Name, key)
		}

		results, ok := schema.Properties["results"].(map[string]any)
		require.True(t, ok, "%s: unexpected results schema shape", tool.Name)
		item, ok := results["items"].(map[string]any)
		require.True(t, ok, "%s: results has no item schema", tool.Name)
		props, ok := item["properties"].(map[string]any)
		require.True(t, ok, "%s: result items have no properties", tool.Name)

		for _, key := range []string{"domain", "status", "available", "for_sale"} {
			assert.Contains(t, props, key, "%s: result item missing %s", tool.Name, key)
		}
	}
}

func TestServer_DescriptionTeachesTheBudget(t *testing.T) {
	isolateConfig(t, "")

	desc := toolNamed(t, listTools(t, newClient(t)), "generate_and_check").Description

	assert.Contains(t, desc, fmt.Sprint(mcpserver.MaxDomainsPerCall), "the cap must be a literal number")
	assert.Contains(t, desc, "dry_run=true")
	assert.Contains(t, desc, "limit=N")
}

func TestServer_PresetEnumIncludesBuiltins(t *testing.T) {
	isolateConfig(t, "")

	tool := toolNamed(t, listTools(t, newClient(t)), "generate_and_check")
	enum := presetEnum(t, tool)

	assert.Contains(t, enum, "popular")
	assert.Contains(t, enum, "tech")
	assert.Contains(t, enum, "startup")
	assert.Contains(t, enum, "all")
}

func TestServer_PresetEnumIncludesCustomPresets(t *testing.T) {
	isolateConfig(t, `
[presets.mystack]
tlds = ["dev", "sh", "io"]
`)

	tool := toolNamed(t, listTools(t, newClient(t)), "generate_and_check")

	assert.Contains(t, presetEnum(t, tool), "mystack")
	assert.Contains(t, tool.InputSchema.Properties["tld_preset"].(map[string]any)["description"],
		"mystack (3)", "the description should advertise how many TLDs the preset expands to")
}

func TestServer_CustomPresetResolvesInACall(t *testing.T) {
	isolateConfig(t, `
[presets.mystack]
tlds = ["dev", "sh"]
`)

	res := callTool(t, newClient(t), "generate_and_check", map[string]any{
		"keywords":   []any{"stripe"},
		"tld_preset": "mystack",
		"dry_run":    true,
	})

	assert.ElementsMatch(t, []string{"stripe.dev", "stripe.sh"}, decode(t, res).Domains)
}

func TestServer_AppliesUserDefaults(t *testing.T) {
	isolateConfig(t, `
[defaults]
tlds = ["io", "ai"]
prefixes = ["get"]
`)

	res := callTool(t, newClient(t), "generate_and_check", map[string]any{
		"keywords": []any{"stripe"},
		"dry_run":  true,
	})

	assert.ElementsMatch(t,
		[]string{"stripe.io", "stripe.ai", "getstripe.io", "getstripe.ai"},
		decode(t, res).Domains)
}

func TestServer_ToolArgsWinOverUserDefaults(t *testing.T) {
	isolateConfig(t, `
[defaults]
tlds = ["io", "ai"]
`)

	res := callTool(t, newClient(t), "generate_and_check", map[string]any{
		"keywords": []any{"stripe"},
		"tlds":     []any{"com"},
		"dry_run":  true,
	})

	assert.Equal(t, []string{"stripe.com"}, decode(t, res).Domains)
}

func presetEnum(t *testing.T, tool mcp.Tool) []string {
	t.Helper()

	prop, ok := tool.InputSchema.Properties["tld_preset"].(map[string]any)
	require.True(t, ok, "tld_preset property missing from schema")

	raw, ok := prop["enum"].([]any)
	require.True(t, ok, "tld_preset has no enum: %v", prop)

	out := make([]string, 0, len(raw))
	for _, v := range raw {
		out = append(out, v.(string))
	}
	return out
}

func TestGenerateAndCheck_RejectsOverBudgetBeforeAnyLookup(t *testing.T) {
	isolateConfig(t, "")
	m := &mockRDAP{err: errNotFound}

	res := callTool(t, newClient(t, withRDAP(m)), "generate_and_check", map[string]any{
		"keywords":   []any{"alpha", "bravo", "charlie", "delta", "echo"},
		"tld_preset": "all",
	})

	require.True(t, res.IsError)
	msg := textOf(t, res)

	assert.Contains(t, msg, fmt.Sprint(mcpserver.MaxDomainsPerCall), "state the cap")
	assert.Contains(t, msg, "Easiest fix", "name the dimension to shrink")
	assert.Contains(t, msg, "limit=", "offer the escape hatch")
	assert.Zero(t, m.calls.Load(), "rejection must happen before any network call")
}

func TestGenerateAndCheck_OverBudgetNamesTheDominantDimension(t *testing.T) {
	isolateConfig(t, "")

	tests := []struct {
		name     string
		args     map[string]any
		contains []string
	}{
		{
			name: "TLD preset dominates",
			args: map[string]any{
				"keywords":   []any{"alpha", "bravo", "charlie", "delta", "echo"},
				"tld_preset": "all",
			},
			contains: []string{`tld_preset "all"`, "shorten the TLD list"},
		},
		{
			name: "keyword list dominates",
			args: map[string]any{
				"keywords": manyKeywords(23),
				"tlds":     []any{"com", "io", "ai", "dev", "app", "co", "net", "org", "xyz", "sh"},
				"prefixes": []any{"get", "use"},
				"suffixes": []any{"ly", "hub"},
			},
			contains: []string{"23 keyword(s)", "split the keywords"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := callTool(t, newClient(t), "generate_and_check", tc.args)
			require.True(t, res.IsError)

			msg := textOf(t, res)
			assert.Contains(t, msg, "Breakdown:", "the factorisation teaches the cost model")
			for _, want := range tc.contains {
				assert.Contains(t, msg, want)
			}
		})
	}
}

func manyKeywords(n int) []any {
	out := make([]any, n)
	for i := range out {
		out[i] = fmt.Sprintf("keyword%d", i)
	}
	return out
}

func TestGenerateAndCheck_LimitBuysAWideSweep(t *testing.T) {
	isolateConfig(t, "")

	res := callTool(t, newClient(t, withRDAP(&mockRDAP{err: errNotFound})), "generate_and_check", map[string]any{
		"keywords":       []any{"alpha", "bravo", "charlie", "delta", "echo"},
		"tld_preset":     "all",
		"only_available": true,
		"limit":          float64(3),
	})

	out := decode(t, res)
	assert.Equal(t, 3, out.Available)
	assert.Len(t, out.Results, 3)
	assert.True(t, out.Truncated)
	assert.Contains(t, out.Note, "limit")
}

func TestCheckDomains_RejectsOverlongList(t *testing.T) {
	isolateConfig(t, "")

	domains := make([]any, mcpserver.MaxDomainsPerCall+1)
	for i := range domains {
		domains[i] = fmt.Sprintf("d%d.com", i)
	}

	res := callTool(t, newClient(t), "check_domains", map[string]any{"domains": domains})

	require.True(t, res.IsError)
	assert.Contains(t, textOf(t, res), fmt.Sprint(mcpserver.MaxDomainsPerCall))
}

func TestGenerateAndCheck_DryRunCostsNothing(t *testing.T) {
	isolateConfig(t, "")
	m := &mockRDAP{err: errNotFound}

	res := callTool(t, newClient(t, withRDAP(m)), "generate_and_check", map[string]any{
		"keywords": []any{"stripe"},
		"tlds":     []any{"com", "io"},
		"prefixes": []any{"get"},
		"dry_run":  true,
	})

	out := decode(t, res)
	assert.True(t, out.DryRun)
	assert.Equal(t, 4, out.Planned)
	assert.ElementsMatch(t,
		[]string{"stripe.com", "stripe.io", "getstripe.com", "getstripe.io"},
		out.Domains)
	assert.Empty(t, out.Results)
	assert.Zero(t, m.calls.Load(), "a dry run must not touch the network")
}

func TestGenerateAndCheck_DryRunWarnsWhenOverBudget(t *testing.T) {
	isolateConfig(t, "")

	res := callTool(t, newClient(t), "generate_and_check", map[string]any{
		"keywords":   []any{"alpha", "bravo", "charlie", "delta", "echo"},
		"tld_preset": "all",
		"dry_run":    true,
	})

	out := decode(t, res)
	assert.Greater(t, out.Planned, mcpserver.MaxDomainsPerCall)
	assert.Contains(t, out.Note, "over the")
}

func TestGenerateAndCheck_MaxDomainLengthShrinksTheCount(t *testing.T) {
	isolateConfig(t, "")

	res := callTool(t, newClient(t), "generate_and_check", map[string]any{
		"keywords":          []any{"stripe"},
		"tlds":              []any{"com"},
		"prefixes":          []any{"averylongprefixindeed"},
		"max_domain_length": float64(12),
		"dry_run":           true,
	})

	assert.Equal(t, []string{"stripe.com"}, decode(t, res).Domains)
}

func TestCheckDomains_ReportsAvailability(t *testing.T) {
	isolateConfig(t, "")

	res := callTool(t, newClient(t, withRDAP(&mockRDAP{err: errNotFound})), "check_domains", map[string]any{
		"domains": []any{"free1.com", "free2.io"},
	})

	out := decode(t, res)
	require.Len(t, out.Results, 2)
	assert.Equal(t, 2, out.Available)
	assert.Equal(t, 2, out.Checked)
	assert.False(t, out.Truncated)

	for _, r := range out.Results {
		assert.Equal(t, mcpserver.StatusAvailable, r.Status)
		require.NotNil(t, r.Available)
		assert.True(t, *r.Available)
	}
}

func TestCheckDomains_ReportsTaken(t *testing.T) {
	isolateConfig(t, "")

	res := callTool(t, newClient(t, withRDAP(&mockRDAP{})), "check_domains", map[string]any{
		"domains": []any{"taken.com"},
	})

	out := decode(t, res)
	require.Len(t, out.Results, 1)
	assert.Equal(t, mcpserver.StatusTaken, out.Results[0].Status)
	require.NotNil(t, out.Results[0].Available)
	assert.False(t, *out.Results[0].Available)
	assert.Equal(t, 1, out.Taken)
}

func TestCheckDomains_FailedLookupIsUnknownNotAvailable(t *testing.T) {
	isolateConfig(t, "")

	res := callTool(t, newClient(t, withRDAP(&mockRDAP{err: fmt.Errorf("boom")})), "check_domains", map[string]any{
		"domains": []any{"mystery.com"},
	})

	out := decode(t, res)
	require.Len(t, out.Results, 1)
	assert.Equal(t, mcpserver.StatusUnknown, out.Results[0].Status)
	assert.Nil(t, out.Results[0].Available, "available must be absent, not false")
	assert.NotEmpty(t, out.Results[0].Error)
	assert.Equal(t, 1, out.Errored)

	// The key must be absent entirely, not false.
	var raw struct {
		Results []map[string]any `json:"results"`
	}
	require.NoError(t, json.Unmarshal([]byte(textOf(t, res)), &raw))
	require.Len(t, raw.Results, 1)
	assert.NotContains(t, raw.Results[0], "available")
}

func TestCheckDomains_RejectsMalformedNames(t *testing.T) {
	isolateConfig(t, "")

	res := callTool(t, newClient(t), "check_domains", map[string]any{
		"domains": []any{"@@@invalid"},
	})

	require.True(t, res.IsError)
	assert.Contains(t, textOf(t, res), "no valid domain names")
}

func TestCheckDomains_SkipsMalformedButChecksTheRest(t *testing.T) {
	isolateConfig(t, "")

	res := callTool(t, newClient(t, withRDAP(&mockRDAP{err: errNotFound})), "check_domains", map[string]any{
		"domains": []any{"@@@invalid", "good.com"},
	})

	out := decode(t, res)
	require.Len(t, out.Results, 1)
	assert.Equal(t, "good.com", out.Results[0].Domain)
	assert.Contains(t, out.Note, "malformed")
}

func TestCheckDomains_RequiresANonEmptyList(t *testing.T) {
	isolateConfig(t, "")

	res := callTool(t, newClient(t), "check_domains", map[string]any{"domains": []any{}})
	assert.True(t, res.IsError)
}

func TestGenerateAndCheck_RequiresKeywords(t *testing.T) {
	isolateConfig(t, "")

	res := callTool(t, newClient(t), "generate_and_check", map[string]any{})
	assert.True(t, res.IsError)
}

func TestGenerateAndCheck_CarriesPermutationMetadata(t *testing.T) {
	isolateConfig(t, "")

	res := callTool(t, newClient(t, withRDAP(&mockRDAP{err: errNotFound})), "generate_and_check", map[string]any{
		"keywords": []any{"stripe"},
		"tlds":     []any{"com"},
		"prefixes": []any{"get"},
		"suffixes": []any{"ly"},
	})

	out := decode(t, res)
	require.NotEmpty(t, out.Results)

	byDomain := map[string]mcpserver.DomainCheck{}
	for _, r := range out.Results {
		byDomain[r.Domain] = r
	}

	full := byDomain["getstripely.com"]
	assert.Equal(t, "stripe", full.Keyword)
	assert.Equal(t, "get", full.Prefix)
	assert.Equal(t, "ly", full.Suffix)
	assert.Equal(t, "com", full.TLD)
}

func TestGenerateAndCheck_OnlyAvailableFiltersResultsButNotCounts(t *testing.T) {
	isolateConfig(t, "")

	res := callTool(t, newClient(t, withRDAP(&mockRDAP{})), "generate_and_check", map[string]any{
		"keywords":       []any{"taken"},
		"tlds":           []any{"com", "io"},
		"only_available": true,
	})

	out := decode(t, res)
	assert.Empty(t, out.Results, "every domain is taken")
	assert.Equal(t, 2, out.Checked, "counts still describe the whole sweep")
	assert.Equal(t, 2, out.Taken)
}

func withForSale(txts ...string) mcpserver.Option {
	return withRDAP(&mockRDAP{}, resolver.WithTXTLookup(
		func(_ context.Context, _ string) ([]string, error) { return txts, nil }))
}

func TestCheckDomains_ForSale(t *testing.T) {
	isolateConfig(t, "")

	res := callTool(t,
		newClient(t, withForSale("v=FORSALE1;fval=USD750", "v=FORSALE1;furi=https://fs.example.com/")),
		"check_domains", map[string]any{
			"domains":        []any{"taken.com"},
			"check_for_sale": true,
		})

	out := decode(t, res)
	require.Len(t, out.Results, 1)
	require.NotNil(t, out.Results[0].ForSale)
	require.Len(t, out.Results[0].ForSale.Prices, 1)
	assert.Equal(t, "USD", out.Results[0].ForSale.Prices[0].Currency)
	assert.Equal(t, "750", out.Results[0].ForSale.Prices[0].Amount)
	assert.Equal(t, 1, out.ForSale)
}

func TestCheckDomains_ForSaleIsOptIn(t *testing.T) {
	isolateConfig(t, "")

	res := callTool(t, newClient(t, withForSale("v=FORSALE1;fval=USD750")),
		"check_domains", map[string]any{"domains": []any{"taken.com"}})

	out := decode(t, res)
	require.Len(t, out.Results, 1)
	assert.Nil(t, out.Results[0].ForSale)
}

func TestGenerateAndCheck_OnlyForSaleImpliesTheLookup(t *testing.T) {
	isolateConfig(t, "")

	res := callTool(t, newClient(t, withForSale("v=FORSALE1;fval=EUR500")),
		"generate_and_check", map[string]any{
			"keywords":      []any{"taken"},
			"tlds":          []any{"com"},
			"only_for_sale": true,
		})

	out := decode(t, res)
	require.Len(t, out.Results, 1, "only_for_sale should turn check_for_sale on by itself")
	require.NotNil(t, out.Results[0].ForSale)
	assert.Equal(t, "EUR", out.Results[0].ForSale.Prices[0].Currency)
}

func TestGenerateAndCheck_OnlyForSaleFiltersOutPlainTakenDomains(t *testing.T) {
	isolateConfig(t, "")

	// No TXT records, so nothing is for sale.
	res := callTool(t, newClient(t, withForSale()), "generate_and_check", map[string]any{
		"keywords":      []any{"taken"},
		"tlds":          []any{"com"},
		"only_for_sale": true,
	})

	out := decode(t, res)
	assert.Empty(t, out.Results)
	assert.Equal(t, 1, out.Checked)
}
