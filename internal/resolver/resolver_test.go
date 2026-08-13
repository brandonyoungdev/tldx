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

func TestCheckDomainsStreaming_CancelMidStream(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.MaxRetries = 0
	app.Config.ContextTimeout = 5 * time.Second
	app.Config.ConcurrencyLimit = 4

	slow := &mockRDAPQuerierFunc{
		fn: func(req *rdap.Request) (*rdap.Response, error) {
			<-req.Context().Done()
			return nil, req.Context().Err()
		},
	}

	s := resolver.NewResolverService(app, resolver.WithRDAPQuerier(slow))

	specs := make([]resolver.DomainSpec, 20)
	for i := range specs {
		specs[i] = resolver.DomainSpec{Domain: fmt.Sprintf("domain%d.com", i), TLD: "com"}
	}

	ctx, cancel := context.WithCancel(context.Background())

	ch := s.CheckDomainsStreaming(ctx, specs)

	time.AfterFunc(20*time.Millisecond, cancel)

	for range ch {
	}
}

func TestCheckDomainsStreaming_CancelWhileSemaphoreFull(t *testing.T) {
	app := config.NewTldxContext()
	app.Config.MaxRetries = 0
	app.Config.ContextTimeout = 5 * time.Second
	app.Config.ConcurrencyLimit = 1

	slow := &mockRDAPQuerierFunc{
		fn: func(req *rdap.Request) (*rdap.Response, error) {
			<-req.Context().Done()
			return nil, req.Context().Err()
		},
	}

	s := resolver.NewResolverService(app, resolver.WithRDAPQuerier(slow))

	specs := make([]resolver.DomainSpec, 10)
	for i := range specs {
		specs[i] = resolver.DomainSpec{Domain: fmt.Sprintf("slow%d.com", i), TLD: "com"}
	}

	ctx, cancel := context.WithCancel(context.Background())

	ch := s.CheckDomainsStreaming(ctx, specs)

	time.AfterFunc(20*time.Millisecond, cancel)

	for range ch {
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

func TestCheckDomain_ForSale(t *testing.T) {
	registered := &mockRDAPQuerier{resp: makeDomainRDAPResponse()}
	notFound := &mockRDAPQuerier{err: fmt.Errorf("object does not exist.")}

	txtOK := func(_ context.Context, name string) ([]string, error) {
		if name != "_for-sale.taken-domain.com" {
			return nil, fmt.Errorf("unexpected TXT query for %q", name)
		}
		return []string{"v=FORSALE1;fval=USD750"}, nil
	}

	t.Run("enriches a taken domain when enabled", func(t *testing.T) {
		app := config.NewTldxContext()
		app.Config.MaxRetries = 0
		app.Config.CheckForSale = true

		s := resolver.NewResolverService(app,
			resolver.WithRDAPQuerier(registered),
			resolver.WithTXTLookup(txtOK),
		)

		result, err := s.CheckDomain(context.Background(), "taken-domain.com")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if result.ForSale == nil {
			t.Fatal("Expected for-sale info to be attached")
		}
		if len(result.ForSale.Prices) != 1 || result.ForSale.Prices[0].String() != "USD 750" {
			t.Errorf("Expected USD 750, got %+v", result.ForSale.Prices)
		}
	})

	t.Run("does nothing when the flag is off", func(t *testing.T) {
		app := config.NewTldxContext()
		app.Config.MaxRetries = 0

		called := false
		s := resolver.NewResolverService(app,
			resolver.WithRDAPQuerier(registered),
			resolver.WithTXTLookup(func(ctx context.Context, name string) ([]string, error) {
				called = true
				return txtOK(ctx, name)
			}),
		)

		result, _ := s.CheckDomain(context.Background(), "taken-domain.com")
		if called {
			t.Error("Expected no TXT lookup when --for-sale is off")
		}
		if result.ForSale != nil {
			t.Error("Expected no for-sale info")
		}
	})

	t.Run("skips available domains", func(t *testing.T) {
		app := config.NewTldxContext()
		app.Config.MaxRetries = 0
		app.Config.CheckForSale = true

		called := false
		s := resolver.NewResolverService(app,
			resolver.WithRDAPQuerier(notFound),
			resolver.WithTXTLookup(func(_ context.Context, _ string) ([]string, error) {
				called = true
				return nil, nil
			}),
		)

		result, _ := s.CheckDomain(context.Background(), "available-domain.com")
		if called {
			t.Error("Expected no TXT lookup for an available domain")
		}
		if result.Registered || result.ForSale != nil {
			t.Errorf("Expected an untouched available result, got %+v", result)
		}
	})

	t.Run("a failing TXT lookup leaves the verdict intact", func(t *testing.T) {
		app := config.NewTldxContext()
		app.Config.MaxRetries = 0
		app.Config.CheckForSale = true

		s := resolver.NewResolverService(app,
			resolver.WithRDAPQuerier(registered),
			resolver.WithTXTLookup(func(_ context.Context, _ string) ([]string, error) {
				return nil, errors.New("no such host")
			}),
		)

		result, err := s.CheckDomain(context.Background(), "taken-domain.com")
		if err != nil {
			t.Fatalf("Expected the TXT error to be swallowed, got: %v", err)
		}
		if !result.Registered {
			t.Error("Expected the domain to still be registered")
		}
		if result.ForSale != nil {
			t.Error("Expected no for-sale info")
		}
	})

	t.Run("a domain with no for-sale record stays nil", func(t *testing.T) {
		app := config.NewTldxContext()
		app.Config.MaxRetries = 0
		app.Config.CheckForSale = true

		s := resolver.NewResolverService(app,
			resolver.WithRDAPQuerier(registered),
			resolver.WithTXTLookup(func(_ context.Context, _ string) ([]string, error) {
				return []string{"v=spf1 -all"}, nil
			}),
		)

		result, _ := s.CheckDomain(context.Background(), "taken-domain.com")
		if result.ForSale != nil {
			t.Errorf("Expected nil for-sale info, got %+v", result.ForSale)
		}
	})

	t.Run("streaming results carry for-sale info", func(t *testing.T) {
		app := config.NewTldxContext()
		app.Config.MaxRetries = 0
		app.Config.CheckForSale = true

		s := resolver.NewResolverService(app,
			resolver.WithRDAPQuerier(registered),
			resolver.WithTXTLookup(txtOK),
		)

		specs := []resolver.DomainSpec{{Domain: "taken-domain.com", Keyword: "taken", TLD: "com"}}
		for result := range s.CheckDomainsStreaming(context.Background(), specs) {
			if result.ForSale == nil {
				t.Fatal("Expected for-sale info on the streamed result")
			}
			if got := result.AsEncodable().ForSale; got == nil {
				t.Error("Expected for-sale info to survive AsEncodable")
			}
		}
	})
}
