package userconfig_test

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/userconfig"
)

// flagsSet builds an isSet func reporting the named flags as explicitly passed.
func flagsSet(names ...string) func(string) bool {
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	return func(flag string) bool { return set[flag] }
}

func TestLoad_ParsesDefaults(t *testing.T) {
	path := withTempConfigPath(t)

	content := `
[defaults]
tlds = ["com", "se", "nu"]
tld_preset = "nordic"
prefixes = ["get"]
suffixes = ["ly"]
max_domain_length = 20
format = "json"
limit = 5
only_available = true
for_sale = true
only_for_sale = true
show_stats = true
no_color = true
verbose = true

[presets.nordic]
tlds = ["se", "nu"]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := userconfig.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	d := cfg.Defaults
	if !reflect.DeepEqual(d.TLDs, []string{"com", "se", "nu"}) {
		t.Errorf("tlds: got %v", d.TLDs)
	}
	if d.TLDPreset != "nordic" {
		t.Errorf("tld_preset: got %q", d.TLDPreset)
	}
	if !reflect.DeepEqual(d.Prefixes, []string{"get"}) {
		t.Errorf("prefixes: got %v", d.Prefixes)
	}
	if !reflect.DeepEqual(d.Suffixes, []string{"ly"}) {
		t.Errorf("suffixes: got %v", d.Suffixes)
	}
	if d.MaxDomainLength == nil || *d.MaxDomainLength != 20 {
		t.Errorf("max_domain_length: got %v", d.MaxDomainLength)
	}
	if d.Format != "json" {
		t.Errorf("format: got %q", d.Format)
	}
	if d.Limit == nil || *d.Limit != 5 {
		t.Errorf("limit: got %v", d.Limit)
	}
	if !d.OnlyAvailable || !d.ShowStats || !d.NoColor || !d.Verbose {
		t.Errorf("expected all bool defaults true, got %+v", d)
	}
	if !d.ForSale || !d.OnlyForSale {
		t.Errorf("expected for-sale defaults true, got %+v", d)
	}
}

func TestLoad_NoDefaultsSection_LeavesZeroDefaults(t *testing.T) {
	path := withTempConfigPath(t)

	if err := os.WriteFile(path, []byte("[presets.x]\ntlds = [\"com\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := userconfig.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !reflect.DeepEqual(cfg.Defaults, userconfig.Defaults{}) {
		t.Errorf("expected zero defaults, got %+v", cfg.Defaults)
	}
}

func TestSave_DefaultsRoundTrip(t *testing.T) {
	withTempConfigPath(t)

	maxLen := 32
	limit := 3
	original := &userconfig.UserConfig{
		Defaults: userconfig.Defaults{
			TLDs:            []string{"com", "se"},
			TLDPreset:       "nordic",
			MaxDomainLength: &maxLen,
			Limit:           &limit,
			OnlyAvailable:   true,
		},
		Presets: map[string]userconfig.PresetEntry{
			"nordic": {TLDs: []string{"se", "nu"}},
		},
	}

	if err := userconfig.Save(original); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := userconfig.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !reflect.DeepEqual(original.Defaults, loaded.Defaults) {
		t.Errorf("round-trip mismatch: saved %+v, loaded %+v", original.Defaults, loaded.Defaults)
	}
}

func TestSave_OmitsUnsetNumericDefaults(t *testing.T) {
	path := withTempConfigPath(t)

	cfg := &userconfig.UserConfig{
		Defaults: userconfig.Defaults{TLDPreset: "popular"},
		Presets:  map[string]userconfig.PresetEntry{},
	}
	if err := userconfig.Save(cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"max_domain_length", "limit", "only_available", "tlds"} {
		if strings.Contains(string(data), key) {
			t.Errorf("expected %q to be omitted from saved config, got:\n%s", key, data)
		}
	}
}

func TestApplyTo_AppliesAllDefaults(t *testing.T) {
	maxLen := 20
	limit := 5
	d := userconfig.Defaults{
		TLDs:            []string{"se", "nu"},
		TLDPreset:       "nordic",
		Prefixes:        []string{"get"},
		Suffixes:        []string{"ly"},
		MaxDomainLength: &maxLen,
		Format:          "json",
		Limit:           &limit,
		OnlyAvailable:   true,
		ForSale:         true,
		OnlyForSale:     true,
		ShowStats:       true,
		NoColor:         true,
		Verbose:         true,
	}

	cfg := &config.TldxConfigOptions{MaxDomainLength: 64, OutputFormat: "text"}
	d.ApplyTo(cfg, flagsSet())

	if !reflect.DeepEqual(cfg.TLDs, []string{"se", "nu"}) {
		t.Errorf("TLDs: got %v", cfg.TLDs)
	}
	if cfg.TLDPreset != "nordic" {
		t.Errorf("TLDPreset: got %q", cfg.TLDPreset)
	}
	if !reflect.DeepEqual(cfg.Prefixes, []string{"get"}) {
		t.Errorf("Prefixes: got %v", cfg.Prefixes)
	}
	if !reflect.DeepEqual(cfg.Suffixes, []string{"ly"}) {
		t.Errorf("Suffixes: got %v", cfg.Suffixes)
	}
	if cfg.MaxDomainLength != 20 {
		t.Errorf("MaxDomainLength: got %d", cfg.MaxDomainLength)
	}
	if cfg.OutputFormat != "json" {
		t.Errorf("OutputFormat: got %q", cfg.OutputFormat)
	}
	if cfg.Limit != 5 {
		t.Errorf("Limit: got %d", cfg.Limit)
	}
	if !cfg.OnlyAvailable || !cfg.ShowStats || !cfg.NoColor || !cfg.Verbose {
		t.Errorf("expected bool options enabled, got %+v", cfg)
	}
	if !cfg.CheckForSale || !cfg.OnlyForSale {
		t.Errorf("expected for-sale options enabled, got %+v", cfg)
	}
}

func TestApplyTo_EmptyDefaultsChangeNothing(t *testing.T) {
	cfg := &config.TldxConfigOptions{MaxDomainLength: 64, OutputFormat: "text"}
	before := *cfg

	userconfig.Defaults{}.ApplyTo(cfg, flagsSet())

	if !reflect.DeepEqual(*cfg, before) {
		t.Errorf("empty defaults mutated config: before %+v, after %+v", before, *cfg)
	}
}

func TestApplyTo_ExplicitFlagsWin(t *testing.T) {
	maxLen := 20
	limit := 5
	d := userconfig.Defaults{
		Prefixes:        []string{"get"},
		Suffixes:        []string{"ly"},
		MaxDomainLength: &maxLen,
		Format:          "json",
		Limit:           &limit,
	}

	cfg := &config.TldxConfigOptions{
		Prefixes:        []string{"my"},
		Suffixes:        []string{"hq"},
		MaxDomainLength: 64,
		OutputFormat:    "csv",
		Limit:           1,
	}
	d.ApplyTo(cfg, flagsSet("prefixes", "suffixes", "max-domain-length", "format", "limit"))

	if !reflect.DeepEqual(cfg.Prefixes, []string{"my"}) {
		t.Errorf("Prefixes should keep flag value, got %v", cfg.Prefixes)
	}
	if !reflect.DeepEqual(cfg.Suffixes, []string{"hq"}) {
		t.Errorf("Suffixes should keep flag value, got %v", cfg.Suffixes)
	}
	if cfg.MaxDomainLength != 64 {
		t.Errorf("MaxDomainLength should keep flag value, got %d", cfg.MaxDomainLength)
	}
	if cfg.OutputFormat != "csv" {
		t.Errorf("OutputFormat should keep flag value, got %q", cfg.OutputFormat)
	}
	if cfg.Limit != 1 {
		t.Errorf("Limit should keep flag value, got %d", cfg.Limit)
	}
}

func TestApplyTo_TLDFlagsSuppressBothTLDDefaults(t *testing.T) {
	d := userconfig.Defaults{TLDs: []string{"se", "nu"}, TLDPreset: "nordic"}

	cfg := &config.TldxConfigOptions{TLDs: []string{"com"}}
	d.ApplyTo(cfg, flagsSet("tlds"))
	if !reflect.DeepEqual(cfg.TLDs, []string{"com"}) {
		t.Errorf("TLDs: got %v, want [com]", cfg.TLDs)
	}
	if cfg.TLDPreset != "" {
		t.Errorf("TLDPreset should stay unset when --tlds is given, got %q", cfg.TLDPreset)
	}

	cfg = &config.TldxConfigOptions{TLDPreset: "tech"}
	d.ApplyTo(cfg, flagsSet("tld-preset"))
	if len(cfg.TLDs) != 0 {
		t.Errorf("TLDs should stay unset when --tld-preset is given, got %v", cfg.TLDs)
	}
	if cfg.TLDPreset != "tech" {
		t.Errorf("TLDPreset: got %q, want tech", cfg.TLDPreset)
	}
}

func TestApplyTo_BoolDefaultsNeverUnsetFlags(t *testing.T) {
	cfg := &config.TldxConfigOptions{
		OnlyAvailable: true, ShowStats: true, NoColor: true, Verbose: true,
		CheckForSale: true, OnlyForSale: true,
	}

	userconfig.Defaults{}.ApplyTo(cfg, flagsSet(
		"only-available", "show-stats", "no-color", "verbose", "for-sale", "only-for-sale"))

	if !cfg.OnlyAvailable || !cfg.ShowStats || !cfg.NoColor || !cfg.Verbose {
		t.Errorf("false defaults should not disable flags, got %+v", cfg)
	}
	if !cfg.CheckForSale || !cfg.OnlyForSale {
		t.Errorf("false defaults should not disable for-sale flags, got %+v", cfg)
	}
}

func TestApplyTo_ForSaleDefaults(t *testing.T) {
	t.Run("for_sale alone enables the lookup", func(t *testing.T) {
		cfg := &config.TldxConfigOptions{}
		userconfig.Defaults{ForSale: true}.ApplyTo(cfg, flagsSet())

		if !cfg.CheckForSale {
			t.Error("expected for_sale to enable CheckForSale")
		}
		if cfg.OnlyForSale {
			t.Error("for_sale must not imply only_for_sale")
		}
	})

	t.Run("only_for_sale alone does not filter without the implication", func(t *testing.T) {
		// ApplyTo only copies the value; the root command derives CheckForSale.
		cfg := &config.TldxConfigOptions{}
		userconfig.Defaults{OnlyForSale: true}.ApplyTo(cfg, flagsSet())

		if !cfg.OnlyForSale {
			t.Error("expected only_for_sale to be copied")
		}
	})
}

func TestApplyTo_CopiesSlicesDefensively(t *testing.T) {
	d := userconfig.Defaults{TLDs: []string{"se"}, Prefixes: []string{"get"}, Suffixes: []string{"ly"}}

	cfg := &config.TldxConfigOptions{}
	d.ApplyTo(cfg, flagsSet())

	cfg.TLDs[0] = "mutated"
	cfg.Prefixes[0] = "mutated"
	cfg.Suffixes[0] = "mutated"

	if d.TLDs[0] != "se" || d.Prefixes[0] != "get" || d.Suffixes[0] != "ly" {
		t.Errorf("defaults were aliased and mutated: %+v", d)
	}
}

func TestApplyTo_NilIsSetTreatsNoFlagsAsPassed(t *testing.T) {
	d := userconfig.Defaults{TLDPreset: "nordic"}

	cfg := &config.TldxConfigOptions{}
	d.ApplyTo(cfg, nil)

	if cfg.TLDPreset != "nordic" {
		t.Errorf("expected defaults to apply with nil isSet, got %q", cfg.TLDPreset)
	}
}

func TestConfigPath_PrefersConfigTOMLOverLegacy(t *testing.T) {
	dir := withFakeUserConfigDir(t)

	legacy := filepath.Join(dir, "tldx", userconfig.LegacyConfigFileName)
	if err := os.MkdirAll(filepath.Dir(legacy), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte("# legacy\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := userconfig.ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error: %v", err)
	}
	if filepath.Base(got) != userconfig.LegacyConfigFileName {
		t.Errorf("expected legacy path, got %s", got)
	}

	preferred := filepath.Join(dir, "tldx", userconfig.ConfigFileName)
	if err := os.WriteFile(preferred, []byte("# new\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err = userconfig.ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error: %v", err)
	}
	if filepath.Base(got) != userconfig.ConfigFileName {
		t.Errorf("expected %s to win, got %s", userconfig.ConfigFileName, got)
	}
}

func TestConfigPath_DefaultsToConfigTOMLWhenNothingExists(t *testing.T) {
	withFakeUserConfigDir(t)

	got, err := userconfig.ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error: %v", err)
	}
	if filepath.Base(got) != userconfig.ConfigFileName {
		t.Errorf("expected %s for a fresh install, got %s", userconfig.ConfigFileName, got)
	}
}

// withFakeUserConfigDir points os.UserConfigDir at a temp dir and clears the
// TLDX_CONFIG override.
func withFakeUserConfigDir(t *testing.T) string {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("os.UserConfigDir uses %AppData% on Windows")
	}

	t.Setenv("TLDX_CONFIG", "")
	tmp := t.TempDir()

	if runtime.GOOS == "darwin" {
		t.Setenv("HOME", tmp)
		return filepath.Join(tmp, "Library", "Application Support")
	}

	t.Setenv("XDG_CONFIG_HOME", tmp)
	return tmp
}
