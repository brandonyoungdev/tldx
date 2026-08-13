package mcpserver_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckDomains_RejectsANonListArgument(t *testing.T) {
	isolateConfig(t, "")

	res := callTool(t, newClient(t), "check_domains", map[string]any{
		"domains": "stripe.com",
	})

	require.True(t, res.IsError, "a bare string is not a domain list")
}

func TestGenerateAndCheck_RejectsAnEmptyKeywordList(t *testing.T) {
	isolateConfig(t, "")

	res := callTool(t, newClient(t), "generate_and_check", map[string]any{
		"keywords": []any{},
	})

	require.True(t, res.IsError)
	assert.Contains(t, textOf(t, res), "must not be empty")
}

func TestGenerateAndCheck_ExplainsWhenNothingSurvivesFiltering(t *testing.T) {
	isolateConfig(t, "")
	m := &mockRDAP{}

	res := callTool(t, newClient(t, withRDAP(m)), "generate_and_check", map[string]any{
		"keywords":          []any{"considerablylongkeyword"},
		"tlds":              []any{"com", "not a tld"},
		"max_domain_length": 5,
	})

	require.True(t, res.IsError)
	msg := textOf(t, res)
	assert.Contains(t, msg, "no domains to check")
	assert.Contains(t, msg, "invalid TLD", "the compile warnings explain part of it")
	assert.Zero(t, m.calls.Load(), "nothing to check means nothing to look up")
}

func TestGenerateAndCheck_ReportsCompileWarnings(t *testing.T) {
	isolateConfig(t, "")

	res := callTool(t, newClient(t, withRDAP(&mockRDAP{})), "generate_and_check", map[string]any{
		"keywords": []any{"stripe"},
		"tlds":     []any{"com", "not a tld"},
	})

	out := decode(t, res)
	assert.Equal(t, 1, out.Checked, "the valid TLD is still checked")
	assert.Contains(t, out.Note, "invalid TLD", "the skipped TLD is reported back")
}

func TestServer_StartsWithAMalformedConfig(t *testing.T) {
	isolateConfig(t, "[defaults\ntlds = ")

	res := callTool(t, newClient(t, withRDAP(&mockRDAP{err: errNotFound})), "check_domains",
		map[string]any{"domains": []any{"stripe.com"}})

	out := decode(t, res)
	assert.Equal(t, 1, out.Available, "an unreadable config falls back to the built-in defaults")
}

func TestServer_OnlyForSaleDefaultImpliesTheLookup(t *testing.T) {
	isolateConfig(t, "[defaults]\nonly_for_sale = true\n")

	res := callTool(t, newClient(t, withForSale("v=FORSALE1;fval=USD250")), "check_domains",
		map[string]any{"domains": []any{"taken.com"}})

	out := decode(t, res)
	require.Len(t, out.Results, 1, "the configured default should turn the TXT lookup on")
	require.NotNil(t, out.Results[0].ForSale)
	assert.Equal(t, "USD", out.Results[0].ForSale.Prices[0].Currency)
}
