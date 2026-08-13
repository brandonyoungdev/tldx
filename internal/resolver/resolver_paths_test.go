package resolver

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/openrdap/rdap"
)

type stubRDAP struct {
	resp *rdap.Response
	err  error
}

func (m *stubRDAP) Do(_ *rdap.Request) (*rdap.Response, error) {
	return m.resp, m.err
}

func TestWithRetry_StopsWhenTheContextIsCancelledDuringBackoff(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.MaxRetries = 3
	app.Config.InitialBackoff = 10 * time.Second
	app.Config.MaxBackoff = time.Minute
	app.Config.BackoffFactor = 2

	svc := NewResolverService(app)
	ctx, cancel := context.WithCancel(context.Background())

	attempts := 0
	_, err := svc.withRetry(ctx, func() (CheckResult, error) {
		attempts++
		cancel()
		return CheckResult{}, errors.New("i/o timeout")
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected the retry to be abandoned after 1 attempt, got %d", attempts)
	}
}

func TestWithRetry_NoAttemptsConfigured(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.MaxRetries = -1

	svc := NewResolverService(app)

	result, err := svc.withRetry(context.Background(), func() (CheckResult, error) {
		t.Fatal("the function must not be called when no attempt is allowed")
		return CheckResult{}, nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result.Registered {
		t.Error("expected an empty result")
	}
}

func TestCheckForSale_DefaultResolverFailsSoftly(t *testing.T) {
	svc := NewResolverService(config.NewTldxContext())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if info := svc.checkForSale(ctx, "example.com"); info != nil {
		t.Errorf("expected a failed lookup to yield no info, got %+v", info)
	}
}

func TestCheckDomain_UnknownStatusOnRDAPFailure(t *testing.T) {
	app := config.NewTldxContext()
	svc := NewResolverService(app, WithRDAPQuerier(&stubRDAP{err: errors.New("boom")}))

	result, err := svc.checkDomain(context.Background(), "example.com")
	if err == nil {
		t.Fatal("expected the RDAP failure to be reported")
	}
	if result.Registered {
		t.Error("a failed lookup must not be reported as registered")
	}
	if !strings.Contains(result.Details, "unknown status") {
		t.Errorf("expected an unknown-status detail, got %q", result.Details)
	}
}

func TestCheckRDAP_CancelledBeforeTheQuery(t *testing.T) {
	app := config.NewTldxContext()
	svc := NewResolverService(app, WithRDAPQuerier(&stubRDAP{resp: &rdap.Response{Object: &rdap.Domain{}}}))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := svc.checkRDAP(ctx, "example.com")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	if !strings.Contains(result.Details, "Context cancelled") {
		t.Errorf("expected a cancellation detail, got %q", result.Details)
	}
}

func TestCheckRDAP_EmptyResponse(t *testing.T) {
	app := config.NewTldxContext()
	// A response carrying a nil domain object: no error, but nothing to read.
	svc := NewResolverService(app, WithRDAPQuerier(&stubRDAP{
		resp: &rdap.Response{Object: (*rdap.Domain)(nil)},
	}))

	result, err := svc.checkRDAP(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Registered {
		t.Error("an empty response must not count as registered")
	}
	if !strings.Contains(result.Details, "No RDAP response") {
		t.Errorf("expected a 'no response' detail, got %q", result.Details)
	}
}

func TestQueryDomainContext_WithoutAnInjectedQuerier(t *testing.T) {
	app := config.NewTldxContext()
	svc := NewResolverService(app)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := svc.QueryDomainContext(ctx, "example.com"); err == nil {
		t.Fatal("expected the cancelled context to fail the RDAP query")
	}
}

func TestCheckWhois_CancelledWhileWaiting(t *testing.T) {
	app := config.NewTldxContext()
	svc := NewResolverService(app, WithWhoisFetcher(func(string, ...string) (string, error) {
		time.Sleep(time.Minute)
		return "", nil
	}))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := svc.checkWhois(ctx, "example.com"); !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestCheckWhois_LookupError(t *testing.T) {
	app := config.NewTldxContext()
	svc := NewResolverService(app, WithWhoisFetcher(func(string, ...string) (string, error) {
		return "", errors.New("dial tcp: connection reset")
	}))

	_, err := svc.checkWhois(context.Background(), "example.com")
	if err == nil || !strings.Contains(err.Error(), "WHOIS lookup error") {
		t.Errorf("expected a WHOIS lookup error, got %v", err)
	}
}

func TestCheckWhois_NotFoundBody(t *testing.T) {
	app := config.NewTldxContext()
	svc := NewResolverService(app, WithWhoisFetcher(func(string, ...string) (string, error) {
		return "No match for \"EXAMPLE.COM\".", nil
	}))

	result, err := svc.checkWhois(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("a 'no match' body is an answer, not an error: %v", err)
	}
	if result.Registered {
		t.Error("expected the domain to be reported as unregistered")
	}
}

func TestCheckWhois_UnparseableBody(t *testing.T) {
	app := config.NewTldxContext()
	svc := NewResolverService(app, WithWhoisFetcher(func(string, ...string) (string, error) {
		return "the registry said something else entirely", nil
	}))

	result, err := svc.checkWhois(context.Background(), "example.com")
	if err == nil || !strings.Contains(err.Error(), "WHOIS parse failed") {
		t.Errorf("expected a parse failure, got %v", err)
	}
	if !strings.Contains(result.Details, "Failed to parse WHOIS") {
		t.Errorf("expected the detail to name the domain, got %q", result.Details)
	}
}

func TestCheckWhois_NoRegistrationData(t *testing.T) {
	app := config.NewTldxContext()
	// Parses cleanly, but carries neither a registrar nor a creation date.
	svc := NewResolverService(app, WithWhoisFetcher(func(string, ...string) (string, error) {
		return "Domain Name: example.com\nUpdated Date: 2020-01-01T00:00:00Z\nDomain Status: ok\n", nil
	}))

	result, err := svc.checkWhois(context.Background(), "example.com")
	if err == nil || !strings.Contains(err.Error(), "no meaningful registration data") {
		t.Errorf("expected the empty record to be rejected, got %v", err)
	}
	if result.Registered {
		t.Error("an empty WHOIS record must not count as registered")
	}
}

func TestCheckDomainsStreaming_DefaultsTheConcurrencyLimit(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.ConcurrencyLimit = 0
	app.Config.ContextTimeout = 5 * time.Second

	svc := NewResolverService(app, WithRDAPQuerier(&stubRDAP{
		err: errors.New("object does not exist."),
	}))

	var got int
	for result := range svc.CheckDomainsStreaming(context.Background(), []DomainSpec{
		{Domain: "one.com"}, {Domain: "two.com"},
	}) {
		if !result.Available {
			t.Errorf("expected %s to be available", result.Domain)
		}
		got++
	}

	if got != 2 {
		t.Errorf("expected 2 results, got %d", got)
	}
}
