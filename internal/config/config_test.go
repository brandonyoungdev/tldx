package config_test

import (
	"testing"
	"time"

	"github.com/brandonyoungdev/tldx/internal/config"
)

func TestNewTldxContext_Defaults(t *testing.T) {
	ctx := config.NewTldxContext()
	if ctx == nil {
		t.Fatal("NewTldxContext returned nil")
	}
	if ctx.Config == nil {
		t.Fatal("Config is nil")
	}

	tests := []struct {
		name string
		got  any
		want any
	}{
		{"MaxRetries", ctx.Config.MaxRetries, 3},
		{"InitialBackoff", ctx.Config.InitialBackoff, 1500 * time.Millisecond},
		{"MaxBackoff", ctx.Config.MaxBackoff, 5 * time.Second},
		{"BackoffFactor", ctx.Config.BackoffFactor, 1.5},
		{"ContextTimeout", ctx.Config.ContextTimeout, 15 * time.Second},
		{"ConcurrencyLimit", ctx.Config.ConcurrencyLimit, 15},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}

func TestNewTldxContext_BooleanDefaults(t *testing.T) {
	ctx := config.NewTldxContext()

	if ctx.Config.Verbose {
		t.Error("Verbose should default to false")
	}
	if ctx.Config.OnlyAvailable {
		t.Error("OnlyAvailable should default to false")
	}
	if ctx.Config.DryRun {
		t.Error("DryRun should default to false")
	}
	if ctx.Config.Regex {
		t.Error("Regex should default to false")
	}
	if ctx.Config.ShowStats {
		t.Error("ShowStats should default to false")
	}
	if ctx.Config.NoColor {
		t.Error("NoColor should default to false")
	}
}

func TestNewTldxContext_SliceDefaults(t *testing.T) {
	ctx := config.NewTldxContext()

	if len(ctx.Config.TLDs) != 0 {
		t.Errorf("TLDs should default to empty, got %v", ctx.Config.TLDs)
	}
	if len(ctx.Config.Prefixes) != 0 {
		t.Errorf("Prefixes should default to empty, got %v", ctx.Config.Prefixes)
	}
	if len(ctx.Config.Suffixes) != 0 {
		t.Errorf("Suffixes should default to empty, got %v", ctx.Config.Suffixes)
	}
}

func TestNewTldxContext_IndependentInstances(t *testing.T) {
	ctx1 := config.NewTldxContext()
	ctx2 := config.NewTldxContext()

	ctx1.Config.MaxRetries = 99
	if ctx2.Config.MaxRetries == 99 {
		t.Error("Modifying one context should not affect another")
	}
}
