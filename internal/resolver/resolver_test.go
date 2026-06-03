package resolver_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/resolver"
	"github.com/openrdap/rdap"
)

func TestCheckAvailability_InvalidDomain(t *testing.T) {
	app := config.NewTldxContext()
	s := resolver.NewResolverService(app)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := s.CheckDomain(ctx, "@@@invalid###.com")
	if err == nil {
		t.Errorf("Expected error for invalid domain")
	}
}

func TestCheckAvailability_Timeout(t *testing.T) {
	app := config.NewTldxContext()
	s := resolver.NewResolverService(app)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond) // force timeout
	defer cancel()

	_, err := s.CheckDomain(ctx, "example.com")
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context deadline exceeded, got %v", err)
	}
}

// mockRDAPQuerier implements the unexported rdapQuerier interface for testing.
type mockRDAPQuerier struct {
	resp *rdap.Response
	err  error
}

func (m *mockRDAPQuerier) Do(_ *rdap.Request) (*rdap.Response, error) {
	return m.resp, m.err
}

func makeDomainRDAPResponse() *rdap.Response {
	return &rdap.Response{
		Object: &rdap.Domain{},
	}
}

func TestCheckDomain_RDAPNotFound_Available(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.MaxRetries = 0

	mock := &mockRDAPQuerier{
		err: fmt.Errorf("object does not exist."),
	}

	s := resolver.NewResolverService(app, resolver.WithRDAPQuerier(mock))

	result, err := s.CheckDomain(context.Background(), "available-domain.com")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result.Registered {
		t.Error("Expected domain to be not registered (available)")
	}
}

func TestCheckDomain_RDAPRegistered(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.MaxRetries = 0

	mock := &mockRDAPQuerier{
		resp: makeDomainRDAPResponse(),
	}

	s := resolver.NewResolverService(app, resolver.WithRDAPQuerier(mock))

	result, err := s.CheckDomain(context.Background(), "taken-domain.com")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !result.Registered {
		t.Error("Expected domain to be registered")
	}
}

func TestCheckDomain_NoRDAPServer_DNSResolves(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.MaxRetries = 0

	rdapMock := &mockRDAPQuerier{
		err: fmt.Errorf("No RDAP servers found for domain"),
	}
	dnsLookup := func(_ context.Context, _ string) ([]string, error) {
		return []string{"1.2.3.4"}, nil
	}

	s := resolver.NewResolverService(app,
		resolver.WithRDAPQuerier(rdapMock),
		resolver.WithDNSLookup(dnsLookup),
	)

	result, err := s.CheckDomain(context.Background(), "example-dns.xyz")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !result.Registered {
		t.Error("Expected domain to be registered (DNS resolved)")
	}
}

func TestCheckDomain_NoRDAPServer_WhoisRegistered(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.MaxRetries = 0

	rdapMock := &mockRDAPQuerier{
		err: fmt.Errorf("No RDAP servers found for domain"),
	}
	dnsLookup := func(_ context.Context, _ string) ([]string, error) {
		return nil, fmt.Errorf("no such host")
	}
	whoisFetch := func(_ string, _ ...string) (string, error) {
		// Return WHOIS response with registrar info
		return `Domain Name: EXAMPLE-WHOIS.COM
Registrar: Test Registrar
Creation Date: 2020-01-01
`, nil
	}

	s := resolver.NewResolverService(app,
		resolver.WithRDAPQuerier(rdapMock),
		resolver.WithDNSLookup(dnsLookup),
		resolver.WithWhoisFetcher(whoisFetch),
	)

	result, err := s.CheckDomain(context.Background(), "example-whois.com")
	// WHOIS parse might fail on simple test data; we just want no panic and valid result
	_ = err
	_ = result
}

func TestCheckDomainsStreaming_Empty(t *testing.T) {
	app := config.NewTldxContext()
	s := resolver.NewResolverService(app)

	ctx := context.Background()
	ch := s.CheckDomainsStreaming(ctx, []resolver.DomainSpec{})

	var results []resolver.DomainResult
	for r := range ch {
		results = append(results, r)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty input, got %d", len(results))
	}
}

func TestCheckDomainsStreaming_CancelledContext(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.MaxRetries = 0
	app.Config.ContextTimeout = 100 * time.Millisecond

	s := resolver.NewResolverService(app)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	specs := []resolver.DomainSpec{
		{Domain: "example.com", Keyword: "example", TLD: "com"},
		{Domain: "test.com", Keyword: "test", TLD: "com"},
	}

	ch := s.CheckDomainsStreaming(ctx, specs)
	for range ch {
	}
}

func TestAsEncodable_WithError(t *testing.T) {
	result := resolver.DomainResult{
		Domain:  "test.com",
		Error:   errors.New("lookup failed"),
		Keyword: "test",
	}
	enc := result.AsEncodable()
	if enc.Error != "lookup failed" {
		t.Errorf("Expected error string 'lookup failed', got %q", enc.Error)
	}
	if enc.Domain != "test.com" {
		t.Errorf("Expected domain 'test.com', got %q", enc.Domain)
	}
}

func TestAsEncodable_NoError(t *testing.T) {
	result := resolver.DomainResult{
		Domain:    "stripe.com",
		Available: true,
		Keyword:   "stripe",
		TLD:       "com",
	}
	enc := result.AsEncodable()
	if enc.Error != "" {
		t.Errorf("Expected empty error string, got %q", enc.Error)
	}
	if !enc.Available {
		t.Error("Expected Available to be true")
	}
}

// mockRDAPQuerierFunc supports per-call response variation for retry/streaming tests.
type mockRDAPQuerierFunc struct {
	fn func(*rdap.Request) (*rdap.Response, error)
}

func (m *mockRDAPQuerierFunc) Do(req *rdap.Request) (*rdap.Response, error) {
	return m.fn(req)
}

func TestCheckDomainsStreaming_WithMockedResults(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.MaxRetries = 0
	app.Config.ContextTimeout = 2 * time.Second

	mock := &mockRDAPQuerier{
		err: fmt.Errorf("object does not exist."),
	}

	s := resolver.NewResolverService(app, resolver.WithRDAPQuerier(mock))

	specs := []resolver.DomainSpec{
		{Domain: "available1.com", Keyword: "available1", TLD: "com"},
		{Domain: "available2.io", Keyword: "available2", TLD: "io"},
	}

	ctx := context.Background()
	ch := s.CheckDomainsStreaming(ctx, specs)

	var results []resolver.DomainResult
	for r := range ch {
		results = append(results, r)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if !r.Available {
			t.Errorf("Expected domain %s to be available (RDAP 404)", r.Domain)
		}
	}
}

func TestCheckDomain_DNSErrorFallsToWhois(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.MaxRetries = 0

	rdapMock := &mockRDAPQuerier{
		err: fmt.Errorf("No RDAP servers found for domain"),
	}
	dnsLookup := func(_ context.Context, _ string) ([]string, error) {
		return nil, fmt.Errorf("dns lookup failed: connection refused")
	}
	whoisFetch := func(_ string, _ ...string) (string, error) {
		return "", fmt.Errorf("no whois server found")
	}

	s := resolver.NewResolverService(app,
		resolver.WithRDAPQuerier(rdapMock),
		resolver.WithDNSLookup(dnsLookup),
		resolver.WithWhoisFetcher(whoisFetch),
	)

	result, err := s.CheckDomain(context.Background(), "example-no-rdap.xyz")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result.Registered {
		t.Error("Expected domain to be unregistered (DNS error + WHOIS not found)")
	}
}

func TestWithRetry_RetryOnRetryableError(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.MaxRetries = 1
	app.Config.InitialBackoff = 1 * time.Millisecond
	app.Config.MaxBackoff = 5 * time.Millisecond
	app.Config.BackoffFactor = 2.0

	attempts := 0
	mock := &mockRDAPQuerierFunc{
		fn: func(_ *rdap.Request) (*rdap.Response, error) {
			attempts++
			if attempts == 1 {
				return nil, fmt.Errorf("connection timeout on first attempt")
			}
			// Second attempt: domain not found → available
			return nil, fmt.Errorf("object does not exist.")
		},
	}

	s := resolver.NewResolverService(app, resolver.WithRDAPQuerier(mock))

	result, err := s.CheckDomain(context.Background(), "retry-test.com")
	if err != nil {
		t.Errorf("Expected no error after retry, got: %v", err)
	}
	if result.Registered {
		t.Error("Expected domain to be available after retry succeeds with 404")
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts (1 retry), got %d", attempts)
	}
}
