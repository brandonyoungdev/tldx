package cmd_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brandonyoungdev/tldx/cmd"
	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/presets"
)

// runWithConfig runs the root command against a config file and returns stdout.
func runWithConfig(t *testing.T, content string, args ...string) string {
	t.Helper()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.toml")
	t.Setenv("TLDX_CONFIG", path)
	if content != "" {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Presets live in a package-level store, so keep runs isolated.
	original := presets.TLDs
	presets.TLDs = presets.NewTypedStore("tld", presets.DefaultTLDPresets)
	t.Cleanup(func() { presets.TLDs = original })

	app := config.NewTldxContext()
	root := cmd.NewRootCmd(app)
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetArgs(args)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := root.ExecuteContext(context.Background())

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r) //nolint:errcheck

	if err != nil {
		t.Fatalf("run %v failed: %v", args, err)
	}
	return buf.String()
}

func TestDefaults_TLDsApplyToRun(t *testing.T) {
	out := runWithConfig(t, `
[defaults]
tlds = ["com", "se", "nu"]
`, "acme", "--dry-run")

	for _, want := range []string{"acme.com", "acme.se", "acme.nu"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %s in output, got:\n%s", want, out)
		}
	}
}

func TestDefaults_TLDPresetApplyToRun(t *testing.T) {
	out := runWithConfig(t, `
[defaults]
tld_preset = "popular"
`, "acme", "--dry-run")

	for _, want := range []string{"acme.com", "acme.io", "acme.dev", "acme.ai"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %s in output, got:\n%s", want, out)
		}
	}
}

func TestDefaults_CustomPresetAsDefault(t *testing.T) {
	out := runWithConfig(t, `
[defaults]
tld_preset = "nordic"

[presets.nordic]
tlds = ["se", "nu", "dk"]
`, "acme", "--dry-run")

	for _, want := range []string{"acme.se", "acme.nu", "acme.dk"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %s in output, got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "acme.com") {
		t.Errorf("expected the default preset to replace the com fallback, got:\n%s", out)
	}
}

func TestDefaults_TLDsFlagOverridesDefaults(t *testing.T) {
	out := runWithConfig(t, `
[defaults]
tlds = ["se", "nu"]
tld_preset = "popular"
`, "acme", "--dry-run", "--tlds", "dk")

	if !strings.Contains(out, "acme.dk") {
		t.Errorf("expected acme.dk, got:\n%s", out)
	}
	for _, unwanted := range []string{"acme.se", "acme.nu", "acme.com"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("expected %s to be suppressed by --tlds, got:\n%s", unwanted, out)
		}
	}
}

func TestDefaults_PresetFlagOverridesDefaultTLDs(t *testing.T) {
	out := runWithConfig(t, `
[defaults]
tlds = ["se", "nu"]
`, "acme", "--dry-run", "--tld-preset", "popular")

	if !strings.Contains(out, "acme.io") {
		t.Errorf("expected preset TLDs, got:\n%s", out)
	}
	if strings.Contains(out, "acme.se") {
		t.Errorf("expected default tlds to be suppressed by --tld-preset, got:\n%s", out)
	}
}

func TestDefaults_PrefixesSuffixesAndMaxLength(t *testing.T) {
	out := runWithConfig(t, `
[defaults]
tlds = ["com"]
prefixes = ["get"]
suffixes = ["ly"]
max_domain_length = 12
`, "acme", "--dry-run")

	if !strings.Contains(out, "getacme.com") {
		t.Errorf("expected prefixed domain, got:\n%s", out)
	}
	if !strings.Contains(out, "acmely.com") {
		t.Errorf("expected suffixed domain, got:\n%s", out)
	}
	// getacmely.com is 13 chars, past the configured limit.
	if strings.Contains(out, "getacmely.com") {
		t.Errorf("expected max_domain_length to filter long permutations, got:\n%s", out)
	}
}

func TestDefaults_NoConfigFileKeepsFlagDefaults(t *testing.T) {
	out := runWithConfig(t, "", "acme", "--dry-run")

	if !strings.Contains(out, "acme.com") {
		t.Errorf("expected the com fallback with no config, got:\n%s", out)
	}
}

func TestDefaults_InvalidMaxDomainLengthIsRejected(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.toml")
	t.Setenv("TLDX_CONFIG", path)
	if err := os.WriteFile(path, []byte("[defaults]\nmax_domain_length = -1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := config.NewTldxContext()
	root := cmd.NewRootCmd(app)
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"acme", "--dry-run"})

	if err := root.ExecuteContext(context.Background()); err == nil {
		t.Fatal("expected an invalid configured max_domain_length to be rejected")
	}
}

// configuredApp runs the root command against a config file and returns the
// resolved options.
func configuredApp(t *testing.T, content string, args ...string) *config.TldxContext {
	t.Helper()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.toml")
	t.Setenv("TLDX_CONFIG", path)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	app := config.NewTldxContext()
	root := cmd.NewRootCmd(app)
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetArgs(args)

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	err := root.ExecuteContext(context.Background())
	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("run %v failed: %v", args, err)
	}
	return app
}

func TestDefaults_ForSaleAppliesToRun(t *testing.T) {
	app := configuredApp(t, "[defaults]\nfor_sale = true\n", "acme", "--dry-run")

	if !app.Config.CheckForSale {
		t.Error("expected for_sale default to enable the lookup")
	}
	if app.Config.OnlyForSale {
		t.Error("for_sale must not imply only_for_sale")
	}
}

func TestDefaults_OnlyForSaleImpliesForSale(t *testing.T) {
	app := configuredApp(t, "[defaults]\nonly_for_sale = true\n", "acme", "--dry-run")

	if !app.Config.OnlyForSale {
		t.Error("expected only_for_sale default to apply")
	}
	if !app.Config.CheckForSale {
		t.Error("expected only_for_sale to imply for_sale")
	}
}

func TestDefaults_ForSaleOffByDefault(t *testing.T) {
	app := configuredApp(t, "[defaults]\ntlds = [\"com\"]\n", "acme", "--dry-run")

	if app.Config.CheckForSale || app.Config.OnlyForSale {
		t.Errorf("for-sale must stay off unless configured, got %+v", app.Config)
	}
}
