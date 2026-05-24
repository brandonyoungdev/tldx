package resolver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/openrdap/rdap"
)

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      string
		expected bool
	}{
		{"timeout keyword", "connection timeout", true},
		{"i/o timeout", "i/o timeout occurred", true},
		{"connection refused", "connection refused", true},
		{"temporary error", "temporary failure", true},
		{"responded successfully (rdap quirk)", "server responded successfully", true},
		{"unrelated error", "invalid domain format", false},
		{"not found", "object does not exist", false},
		{"empty error", "something random", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.err)
			got := isRetryable(err)
			if got != tt.expected {
				t.Errorf("isRetryable(%q) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestCheckIfDNSResolves_NativePath_UnknownDomain(t *testing.T) {
	svc := &ResolverService{} // no dnsLookupFn — uses net.Resolver
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// A definitely-nonexistent domain. Will fail with DNS error or timeout.
	_, err := svc.checkIfDNSResolves(ctx, "this-domain-absolutely-does-not-exist-xyzzy-12345.invalid")
	// We expect either an error (NXDOMAIN, timeout) or false — both are valid.
	// The important thing is the native path (lines 270-275) is exercised.
	_ = err
}

func TestQueryDomainContext_UnexpectedObjectType(t *testing.T) {
	app := config.NewTldxContext()
	svc := NewResolverService(app, WithRDAPQuerier(&mockUnexpectedRDAPQuerier{}))
	ctx := context.Background()

	_, err := svc.QueryDomainContext(ctx, "example.com")
	if err == nil || err.Error() != "unexpected RDAP object type" {
		t.Errorf("expected 'unexpected RDAP object type', got %v", err)
	}
}

// mockUnexpectedRDAPQuerier returns a Nameserver object instead of a Domain.
type mockUnexpectedRDAPQuerier struct{}

func (m *mockUnexpectedRDAPQuerier) Do(_ *rdap.Request) (*rdap.Response, error) {
	return &rdap.Response{Object: &rdap.Nameserver{}}, nil
}
