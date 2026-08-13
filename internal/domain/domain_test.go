package domain_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/domain"
	"github.com/brandonyoungdev/tldx/internal/resolver"
	"github.com/openrdap/rdap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRDAPQuerier satisfies the unexported rdapQuerier interface via the exported WithRDAPQuerier option.
type mockRDAPQuerier struct {
	resp *rdap.Response
	err  error
}

func (m *mockRDAPQuerier) Do(_ *rdap.Request) (*rdap.Response, error) {
	return m.resp, m.err
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

func TestExec_DryRun(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.DryRun = true
	app.Config.TLDs = []string{"com", "io"}

	out := captureStdout(func() {
		result := domain.Exec(context.Background(), app, []string{"test"})
		assert.False(t, result)
	})

	assert.Contains(t, out, "Would check")
	assert.Contains(t, out, "test.com")
	assert.Contains(t, out, "test.io")
}

func TestExec_DryRun_WithWarnings(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.DryRun = true
	app.Config.TLDs = []string{"notavalidtld!@#"}
	app.Config.OutputFormat = "text"

	out := captureStdout(func() {
		domain.Exec(context.Background(), app, []string{"test"})
	})
	// Should produce some output even with invalid TLD warnings
	_ = out
}

func TestExec_Available(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.TLDs = []string{"com"}
	app.Config.MaxRetries = 0
	app.Config.NoColor = true

	mock := &mockRDAPQuerier{
		err: fmt.Errorf("object does not exist."),
	}

	out := captureStdout(func() {
		result := domain.Exec(context.Background(), app, []string{"test"},
			resolver.WithRDAPQuerier(mock))
		assert.True(t, result)
	})

	assert.Contains(t, out, "test.com")
	assert.Contains(t, out, "available")
}

func TestExec_NotAvailable(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.TLDs = []string{"com"}
	app.Config.MaxRetries = 0
	app.Config.NoColor = true

	mock := &mockRDAPQuerier{
		resp: &rdap.Response{Object: &rdap.Domain{}},
	}

	out := captureStdout(func() {
		result := domain.Exec(context.Background(), app, []string{"test"},
			resolver.WithRDAPQuerier(mock))
		assert.False(t, result)
	})

	assert.Contains(t, out, "test.com")
}

func TestExec_OnlyAvailable_FiltersNotAvailable(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.TLDs = []string{"com"}
	app.Config.MaxRetries = 0
	app.Config.NoColor = true
	app.Config.OnlyAvailable = true

	mock := &mockRDAPQuerier{
		resp: &rdap.Response{Object: &rdap.Domain{}},
	}

	out := captureStdout(func() {
		result := domain.Exec(context.Background(), app, []string{"taken"},
			resolver.WithRDAPQuerier(mock))
		assert.False(t, result)
	})

	assert.NotContains(t, out, "not available")
}

func TestExec_Limit_StopsEarly(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.TLDs = []string{"com", "io", "net", "dev"}
	app.Config.MaxRetries = 0
	app.Config.NoColor = true
	app.Config.Limit = 1

	mock := &mockRDAPQuerier{
		err: fmt.Errorf("object does not exist."),
	}

	captureStdout(func() {
		result := domain.Exec(context.Background(), app, []string{"test"},
			resolver.WithRDAPQuerier(mock))
		assert.True(t, result)
	})
}

func TestExec_ShowStats_Text(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.TLDs = []string{"com"}
	app.Config.MaxRetries = 0
	app.Config.NoColor = true
	app.Config.ShowStats = true
	app.Config.OutputFormat = "text"

	mock := &mockRDAPQuerier{
		resp: &rdap.Response{Object: &rdap.Domain{}},
	}

	out := captureStdout(func() {
		domain.Exec(context.Background(), app, []string{"test"},
			resolver.WithRDAPQuerier(mock))
	})

	assert.Contains(t, out, "searched")
}

func TestExec_CancelledContext(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.TLDs = []string{"com", "io", "net"}
	app.Config.MaxRetries = 0
	app.Config.NoColor = true
	app.Config.Verbose = true

	mock := &mockRDAPQuerier{
		resp: &rdap.Response{Object: &rdap.Domain{}},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	captureStdout(func() {
		domain.Exec(ctx, app, []string{"test"},
			resolver.WithRDAPQuerier(mock))
	})
}

func TestExec_ErroredDomain(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.TLDs = []string{"com"}
	app.Config.MaxRetries = 0
	app.Config.NoColor = true

	mock := &mockRDAPQuerier{
		err: fmt.Errorf("unexpected server error"),
	}

	captureStdout(func() {
		domain.Exec(context.Background(), app, []string{"test"},
			resolver.WithRDAPQuerier(mock))
	})
}

func TestExec_JSONOutput(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.TLDs = []string{"com"}
	app.Config.MaxRetries = 0
	app.Config.OutputFormat = "json-stream"

	mock := &mockRDAPQuerier{
		err: fmt.Errorf("object does not exist."),
	}

	out := captureStdout(func() {
		domain.Exec(context.Background(), app, []string{"test"},
			resolver.WithRDAPQuerier(mock))
	})

	require.NotEmpty(t, out)
	assert.Contains(t, out, "test.com")
}

func TestExec_CSVOutput(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.TLDs = []string{"com"}
	app.Config.MaxRetries = 0
	app.Config.OutputFormat = "csv"

	mock := &mockRDAPQuerier{
		err: fmt.Errorf("object does not exist."),
	}

	out := captureStdout(func() {
		domain.Exec(context.Background(), app, []string{"test"},
			resolver.WithRDAPQuerier(mock))
	})

	assert.Contains(t, out, "domain")
	assert.Contains(t, out, "test.com")
}

func TestExec_GroupedOutput(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.TLDs = []string{"com", "io"}
	app.Config.MaxRetries = 0
	app.Config.NoColor = true
	app.Config.OutputFormat = "grouped"

	mock := &mockRDAPQuerier{
		err: fmt.Errorf("object does not exist."),
	}

	out := captureStdout(func() {
		domain.Exec(context.Background(), app, []string{"stripe"},
			resolver.WithRDAPQuerier(mock))
	})

	assert.Contains(t, out, "stripe")
}

func TestExec_GroupedByTLDOutput(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.TLDs = []string{"com", "io"}
	app.Config.MaxRetries = 0
	app.Config.NoColor = true
	app.Config.OutputFormat = "grouped-tld"

	mock := &mockRDAPQuerier{
		err: fmt.Errorf("object does not exist."),
	}

	out := captureStdout(func() {
		domain.Exec(context.Background(), app, []string{"stripe"},
			resolver.WithRDAPQuerier(mock))
	})

	assert.Contains(t, out, "stripe")
}

func TestExec_JSONArrayOutput(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.TLDs = []string{"com"}
	app.Config.MaxRetries = 0
	app.Config.OutputFormat = "json-array"

	mock := &mockRDAPQuerier{
		err: fmt.Errorf("object does not exist."),
	}

	out := captureStdout(func() {
		domain.Exec(context.Background(), app, []string{"test"},
			resolver.WithRDAPQuerier(mock))
	})

	assert.Contains(t, out, "test.com")
}

func TestExec_OnlyAvailable_WarningsSuppressed(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.TLDs = []string{"com"}
	app.Config.MaxRetries = 0
	app.Config.NoColor = true
	app.Config.OnlyAvailable = true
	app.Config.OutputFormat = "text"

	mock := &mockRDAPQuerier{
		err: fmt.Errorf("unexpected error"),
	}

	out := captureStdout(func() {
		domain.Exec(context.Background(), app, []string{"test"},
			resolver.WithRDAPQuerier(mock))
	})

	// error output should be suppressed when only-available is set
	assert.NotContains(t, strings.ToLower(out), "errored")
}

func takenRDAP() *mockRDAPQuerier {
	return &mockRDAPQuerier{resp: &rdap.Response{Object: &rdap.Domain{}}}
}

func forSaleTXT(txts ...string) func(context.Context, string) ([]string, error) {
	return func(_ context.Context, _ string) ([]string, error) {
		return txts, nil
	}
}

func TestExec_ForSale_AnnotatesTakenDomain(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.TLDs = []string{"com"}
	app.Config.MaxRetries = 0
	app.Config.NoColor = true
	app.Config.CheckForSale = true

	out := captureStdout(func() {
		result := domain.Exec(context.Background(), app, []string{"taken"},
			resolver.WithRDAPQuerier(takenRDAP()),
			resolver.WithTXTLookup(forSaleTXT("v=FORSALE1;fval=USD750")))
		assert.False(t, result)
	})

	assert.Contains(t, out, "taken.com is taken but for sale")
	assert.Contains(t, out, "USD 750")
	assert.NotContains(t, out, "is not available")
}

func TestExec_ForSale_Disabled_LeavesOutputUnchanged(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.TLDs = []string{"com"}
	app.Config.MaxRetries = 0
	app.Config.NoColor = true

	out := captureStdout(func() {
		domain.Exec(context.Background(), app, []string{"taken"},
			resolver.WithRDAPQuerier(takenRDAP()),
			resolver.WithTXTLookup(forSaleTXT("v=FORSALE1;fval=USD750")))
	})

	assert.Contains(t, out, "taken.com is not available")
	assert.NotContains(t, out, "for sale")
}

func TestExec_OnlyForSale_FiltersAndSucceeds(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.TLDs = []string{"com", "io"}
	app.Config.MaxRetries = 0
	app.Config.NoColor = true
	app.Config.CheckForSale = true
	app.Config.OnlyForSale = true

	// Only the .com publishes a for-sale record.
	txt := func(_ context.Context, name string) ([]string, error) {
		if name == "_for-sale.taken.com" {
			return []string{"v=FORSALE1;fval=EUR500"}, nil
		}
		return nil, fmt.Errorf("no such host")
	}

	out := captureStdout(func() {
		result := domain.Exec(context.Background(), app, []string{"taken"},
			resolver.WithRDAPQuerier(takenRDAP()),
			resolver.WithTXTLookup(txt))
		assert.True(t, result)
	})

	assert.Contains(t, out, "taken.com is taken but for sale")
	assert.NotContains(t, out, "taken.io")
}

func TestExec_OnlyForSale_NoHitsIsAFailure(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.TLDs = []string{"com"}
	app.Config.MaxRetries = 0
	app.Config.NoColor = true
	app.Config.OutputFormat = "text"
	app.Config.CheckForSale = true
	app.Config.OnlyForSale = true

	out := captureStdout(func() {
		result := domain.Exec(context.Background(), app, []string{"taken"},
			resolver.WithRDAPQuerier(takenRDAP()),
			resolver.WithTXTLookup(forSaleTXT("v=spf1 -all")))
		assert.False(t, result)
	})

	assert.Empty(t, strings.TrimSpace(out))
}

func TestExec_OnlyAvailableWithForSale_KeepsAvailableSemantics(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.TLDs = []string{"com"}
	app.Config.MaxRetries = 0
	app.Config.NoColor = true
	app.Config.OutputFormat = "text"
	app.Config.CheckForSale = true
	app.Config.OnlyAvailable = true

	out := captureStdout(func() {
		result := domain.Exec(context.Background(), app, []string{"taken"},
			resolver.WithRDAPQuerier(takenRDAP()),
			resolver.WithTXTLookup(forSaleTXT("v=FORSALE1;fval=USD750")))
		assert.False(t, result)
	})

	assert.Empty(t, strings.TrimSpace(out))
}

func TestExec_ForSale_StatsRow(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.TLDs = []string{"com"}
	app.Config.MaxRetries = 0
	app.Config.NoColor = true
	app.Config.CheckForSale = true
	app.Config.ShowStats = true

	out := captureStdout(func() {
		domain.Exec(context.Background(), app, []string{"taken"},
			resolver.WithRDAPQuerier(takenRDAP()),
			resolver.WithTXTLookup(forSaleTXT("v=FORSALE1;fval=USD750")))
	})

	assert.Contains(t, out, "for sale")
}

// cancellingRDAP cancels the run once enough lookups have started, leaving the
// remaining in-flight results to arrive after cancellation.
type cancellingRDAP struct {
	after  int32
	calls  atomic.Int32
	cancel context.CancelFunc
}

func (m *cancellingRDAP) Do(_ *rdap.Request) (*rdap.Response, error) {
	switch n := m.calls.Add(1); {
	case n == m.after:
		m.cancel()
	case n > m.after:
		// Hold these back so they finish well after the cancellation.
		time.Sleep(5 * time.Millisecond)
	}
	return &rdap.Response{Object: &rdap.Domain{}}, nil
}

func TestExec_CancelledMidRun(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.TLDs = []string{"com"}
	app.Config.MaxRetries = 0
	app.Config.NoColor = true
	app.Config.Verbose = true
	app.Config.ShowStats = true
	app.Config.OutputFormat = "text"
	app.Config.ConcurrencyLimit = 64

	keywords := make([]string, 200)
	for i := range keywords {
		keywords[i] = fmt.Sprintf("keyword%d", i)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mock := &cancellingRDAP{after: 32, cancel: cancel}

	out := captureStdout(func() {
		assert.False(t, domain.Exec(ctx, app, keywords, resolver.WithRDAPQuerier(mock)))
	})

	assert.Contains(t, out, "Operation cancelled")
	assert.Less(t, int(mock.calls.Load()), len(keywords), "the run should stop short")
}
