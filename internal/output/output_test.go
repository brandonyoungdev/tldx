package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/output"
	"github.com/brandonyoungdev/tldx/internal/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func availableResult(domain, keyword, prefix, suffix, tld string) resolver.DomainResult {
	return resolver.DomainResult{
		Domain:    domain,
		Available: true,
		Keyword:   keyword,
		Prefix:    prefix,
		Suffix:    suffix,
		TLD:       tld,
	}
}

func TestJsonArrayOutput_BasicArray(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.ShowStats = false

	var buf bytes.Buffer
	w := output.NewJsonArrayOutput(&buf, app)
	w.Write(availableResult("stripe.com", "stripe", "", "", "com"))
	w.Write(availableResult("stripe.io", "stripe", "", "", "io"))
	w.Flush()

	var results []map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &results))
	require.Len(t, results, 2)
	assert.Equal(t, "stripe.com", results[0]["domain"])
	assert.Equal(t, "stripe", results[0]["keyword"])
	assert.Equal(t, "com", results[0]["tld"])
}

func TestJsonArrayOutput_WithStats(t *testing.T) {
	// Reset global stat
	output.Stat = output.Stats{Total: 2, Available: 1, NotAvailable: 1}

	app := config.NewTldxContext()
	app.Config.ShowStats = true

	var buf bytes.Buffer
	w := output.NewJsonArrayOutput(&buf, app)
	w.Write(availableResult("stripe.com", "stripe", "", "", "com"))
	w.Flush()

	var payload map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &payload))

	assert.Contains(t, payload, "results", "stats output should have a results key")
	assert.Contains(t, payload, "stats", "stats output should have a stats key")

	stats, ok := payload["stats"].(map[string]any)
	require.True(t, ok)
	assert.EqualValues(t, 2, stats["Total"])
	assert.EqualValues(t, 1, stats["Available"])
}

func TestJsonArrayOutput_StatsOmittedWhenDisabled(t *testing.T) {
	output.Stat = output.Stats{Total: 5, Available: 3}

	app := config.NewTldxContext()
	app.Config.ShowStats = false

	var buf bytes.Buffer
	w := output.NewJsonArrayOutput(&buf, app)
	w.Write(availableResult("a.com", "a", "", "", "com"))
	w.Flush()

	var results []any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &results))
}

func TestEncodableDomainResult_AllFields(t *testing.T) {
	r := resolver.DomainResult{
		Domain:    "getstripely.io",
		Available: true,
		Keyword:   "stripe",
		Prefix:    "get",
		Suffix:    "ly",
		TLD:       "io",
	}

	enc := r.AsEncodable()
	b, err := json.Marshal(enc)
	require.NoError(t, err)
	s := string(b)

	assert.Contains(t, s, `"keyword":"stripe"`)
	assert.Contains(t, s, `"prefix":"get"`)
	assert.Contains(t, s, `"suffix":"ly"`)
	assert.Contains(t, s, `"tld":"io"`)
	assert.Contains(t, s, `"domain":"getstripely.io"`)
	assert.Contains(t, s, `"available":true`)
}

func TestEncodableDomainResult_EmptyFieldsOmitted(t *testing.T) {
	r := resolver.DomainResult{
		Domain:    "stripe.com",
		Available: false,
	}

	b, err := json.Marshal(r.AsEncodable())
	require.NoError(t, err)
	s := string(b)

	assert.NotContains(t, s, `"keyword"`)
	assert.NotContains(t, s, `"prefix"`)
	assert.NotContains(t, s, `"suffix"`)
	assert.NotContains(t, s, `"tld"`)
}

func TestCSVOutput_HeadersIncludeMetadata(t *testing.T) {
	assert.NotPanics(t, func() {
		w := output.NewCSVOutput()
		w.Write(availableResult("stripe.com", "stripe", "", "", "com"))
	})
}

func TestStyleService_NoColorWhenNoColorEnvSet(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	app := config.NewTldxContext()
	svc := output.NewStyleService(app)
	result := svc.Styled("hello", "10")
	assert.Equal(t, "hello", strings.TrimSpace(result))
}

func TestStyleService_NoColorFlag(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.NoColor = true
	svc := output.NewStyleService(app)
	result := svc.Styled("hello", "10")
	assert.Equal(t, "hello", strings.TrimSpace(result))
}
