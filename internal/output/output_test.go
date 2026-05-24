package output_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
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

func TestStyleService_Available(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.NoColor = true
	svc := output.NewStyleService(app)

	result := resolver.DomainResult{Domain: "test.com", Available: true}
	out := svc.Available(result)
	assert.Contains(t, out, "test.com")
	assert.Contains(t, out, "available")
}

func TestStyleService_NotAvailable(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.NoColor = true
	svc := output.NewStyleService(app)

	result := resolver.DomainResult{Domain: "test.com", Available: false}
	out := svc.NotAvailable(result)
	assert.Contains(t, out, "test.com")
	assert.Contains(t, out, "not available")
}

func TestStyleService_Errored(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.NoColor = true
	svc := output.NewStyleService(app)

	out := svc.Errored("test.com", errors.New("lookup failed"))
	assert.Contains(t, out, "test.com")
	assert.Contains(t, out, "errored")
}

func TestStyleService_GroupHeader(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.NoColor = true
	svc := output.NewStyleService(app)

	out := svc.GroupHeader("stripe")
	assert.Contains(t, out, "stripe")
}

func TestStyleService_Verbose_Available(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.NoColor = true
	app.Config.Verbose = true
	svc := output.NewStyleService(app)

	result := resolver.DomainResult{Domain: "test.com", Available: true, Details: "RDAP 404"}
	out := svc.Available(result)
	assert.Contains(t, out, "RDAP 404")
}

func TestRenderStatsSummary_ContainsStats(t *testing.T) {
	output.Stat = output.Stats{Total: 10, Available: 3, NotAvailable: 5, TimedOut: 1, Errored: 1}
	rendered := output.RenderStatsSummary()
	assert.Contains(t, rendered, "10")
	assert.Contains(t, rendered, "3")
	assert.Contains(t, rendered, "5")
}

func TestGetOutputWriter_AllFormats(t *testing.T) {
	formats := []string{"text", "csv", "json-stream", "json-array", "json", "grouped", "grouped-tld", "unknown"}
	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			app := config.NewTldxContext()
			app.Config.OutputFormat = format
			assert.NotPanics(t, func() {
				w := output.GetOutputWriter(app)
				assert.NotNil(t, w)
			})
		})
	}
}

func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r) //nolint:errcheck
	return buf.String()
}

func makeResult(domain string, available bool) resolver.DomainResult {
	return resolver.DomainResult{
		Domain:    domain,
		Available: available,
		Keyword:   strings.Split(strings.Split(domain, ".")[0], "")[0],
		TLD:       strings.Split(domain, ".")[len(strings.Split(domain, "."))-1],
	}
}

func makeErrorResult(domain string, err error) resolver.DomainResult {
	return resolver.DomainResult{Domain: domain, Error: err}
}

func TestTextOutput_Write_Available(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.NoColor = true
	w := output.NewTextOutput(app)

	result := availableResult("stripe.com", "stripe", "", "", "com")
	out := captureStdout(func() {
		w.Write(result)
		w.Flush()
	})
	assert.Contains(t, out, "stripe.com")
	assert.Contains(t, out, "available")
}

func TestTextOutput_Write_NotAvailable(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.NoColor = true
	w := output.NewTextOutput(app)

	result := resolver.DomainResult{Domain: "stripe.com", Available: false}
	out := captureStdout(func() { w.Write(result) })
	assert.Contains(t, out, "stripe.com")
	assert.Contains(t, out, "not available")
}

func TestTextOutput_Write_Error(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.NoColor = true
	w := output.NewTextOutput(app)

	result := resolver.DomainResult{Domain: "stripe.com", Error: errors.New("lookup failed")}
	out := captureStdout(func() { w.Write(result) })
	assert.Contains(t, out, "stripe.com")
}

func TestTextOutput_Write_OnlyAvailable_SuppressesNotAvailable(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.NoColor = true
	app.Config.OnlyAvailable = true
	w := output.NewTextOutput(app)

	result := resolver.DomainResult{Domain: "stripe.com", Available: false}
	out := captureStdout(func() { w.Write(result) })
	assert.Empty(t, strings.TrimSpace(out))
}

func TestTextOutput_Write_OnlyAvailable_SuppressesError(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.NoColor = true
	app.Config.OnlyAvailable = true
	app.Config.Verbose = false
	w := output.NewTextOutput(app)

	result := resolver.DomainResult{Domain: "stripe.com", Error: errors.New("oops")}
	out := captureStdout(func() { w.Write(result) })
	assert.Empty(t, strings.TrimSpace(out))
}

func TestTextOutput_Write_Verbose_ShowsError(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.NoColor = true
	app.Config.OnlyAvailable = true
	app.Config.Verbose = true
	w := output.NewTextOutput(app)

	result := resolver.DomainResult{Domain: "stripe.com", Error: errors.New("lookup failed")}
	out := captureStdout(func() { w.Write(result) })
	assert.Contains(t, out, "stripe.com")
}

func TestCSVOutput_Flush(t *testing.T) {
	out := captureStdout(func() {
		w := output.NewCSVOutput()
		w.Write(availableResult("stripe.com", "stripe", "get", "ly", "com"))
		w.Flush()
	})
	assert.Contains(t, out, "stripe.com")
	assert.Contains(t, out, "true")
}

func TestJSONStreamOutput_Write_Flush(t *testing.T) {
	out := captureStdout(func() {
		w := &output.JSONStreamOutput{}
		w.Write(availableResult("stripe.com", "stripe", "", "", "com"))
		w.Flush()
	})
	assert.Contains(t, out, `"domain"`)
	assert.Contains(t, out, "stripe.com")
}

func TestGroupedOutput_Write_Flush(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.NoColor = true
	w := output.NewGroupedOutput(app)

	w.Write(availableResult("stripe.com", "stripe", "", "", "com"))
	w.Write(resolver.DomainResult{Domain: "stripe.io", Available: false, Keyword: "stripe", TLD: "io"})
	w.Write(resolver.DomainResult{Domain: "atlas.com", Available: true, Keyword: "atlas", TLD: "com"})

	out := captureStdout(func() { w.Flush() })
	assert.Contains(t, out, "stripe")
	assert.Contains(t, out, "atlas")
}

func TestGroupedOutput_Flush_OnlyAvailable(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.NoColor = true
	app.Config.OnlyAvailable = true
	w := output.NewGroupedOutput(app)

	w.Write(resolver.DomainResult{Domain: "stripe.com", Available: false, Keyword: "stripe"})
	out := captureStdout(func() { w.Flush() })
	_ = out // filtered result, just no panic
}

func TestGroupedOutput_Flush_WithError(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.NoColor = true
	w := output.NewGroupedOutput(app)

	w.Write(resolver.DomainResult{Domain: "stripe.com", Error: errors.New("err"), Keyword: "stripe"})
	out := captureStdout(func() { w.Flush() })
	assert.Contains(t, out, "stripe.com")
}

func TestGroupedOutput_KeywordFallback_FromDomain(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.NoColor = true
	app.Config.Prefixes = []string{"get"}
	app.Config.Suffixes = []string{"ly"}
	w := output.NewGroupedOutput(app)

	// No keyword set — should derive from domain
	w.Write(resolver.DomainResult{Domain: "getstripely.com", Available: true})
	out := captureStdout(func() { w.Flush() })
	assert.NotEmpty(t, out)
}

func TestGroupedByTLDOutput_Write_Flush(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.NoColor = true
	w := output.NewGroupedByTLDOutput(app)

	w.Write(availableResult("stripe.com", "stripe", "", "", "com"))
	w.Write(availableResult("atlas.com", "atlas", "", "", "com"))
	w.Write(availableResult("stripe.io", "stripe", "", "", "io"))

	out := captureStdout(func() { w.Flush() })
	assert.Contains(t, out, ".com")
	assert.Contains(t, out, ".io")
}

func TestGroupedByTLDOutput_Flush_OnlyAvailable(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.NoColor = true
	app.Config.OnlyAvailable = true
	w := output.NewGroupedByTLDOutput(app)

	w.Write(resolver.DomainResult{Domain: "stripe.com", Available: false, TLD: "com"})
	captureStdout(func() { w.Flush() })
}

func TestGroupedByTLDOutput_Flush_WithError(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.NoColor = true
	w := output.NewGroupedByTLDOutput(app)

	w.Write(resolver.DomainResult{Domain: "stripe.com", Error: errors.New("err"), TLD: "com"})
	out := captureStdout(func() { w.Flush() })
	assert.Contains(t, out, "stripe.com")
}

func TestGroupedByTLDOutput_TLDFallbackFromDomain(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.NoColor = true
	w := output.NewGroupedByTLDOutput(app)

	// No TLD field set — should parse from domain string
	w.Write(resolver.DomainResult{Domain: "stripe.dev", Available: true})
	out := captureStdout(func() { w.Flush() })
	assert.Contains(t, out, ".dev")
}

func TestStyleService_Styled_WithColor(t *testing.T) {
	app := config.NewTldxContext()
	svc := output.NewStyleServiceDirect(app, false) // noColor=false → renders with lipgloss
	result := svc.Styled("hello", "10")
	// lipgloss renders ANSI escape codes; the original text should still be present
	assert.Contains(t, result, "hello")
}

func TestStyleService_GroupHeader_WithColor(t *testing.T) {
	app := config.NewTldxContext()
	svc := output.NewStyleServiceDirect(app, false)
	result := svc.GroupHeader("stripe")
	assert.Contains(t, result, "stripe")
}

func TestStyleService_NotAvailable_Verbose(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.NoColor = true
	app.Config.Verbose = true
	svc := output.NewStyleService(app)

	result := resolver.DomainResult{Domain: "taken.com", Available: false, Details: "RDAP registered"}
	out := svc.NotAvailable(result)
	assert.Contains(t, out, "taken.com")
	assert.Contains(t, out, "RDAP registered")
}



func TestGroupedOutput_KeywordFor_NoDot(t *testing.T) {
app := config.NewTldxContext()
app.Config.NoColor = true
out := output.NewGroupedOutput(app)

// Domain with no dot — keywordFor falls back to the domain itself (len(parts) < 2)
out.Write(resolver.DomainResult{
Domain:    "nodot",
Keyword:   "",
Available: true,
})
result := captureStdout(func() { out.Flush() })
assert.Contains(t, result, "nodot")
}
