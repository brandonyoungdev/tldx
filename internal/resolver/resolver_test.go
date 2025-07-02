package resolver_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/resolver"
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
