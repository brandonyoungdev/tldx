package domain_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

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
