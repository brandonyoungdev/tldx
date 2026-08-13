package cmd_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/brandonyoungdev/tldx/internal/userconfig"
)

func TestPresetDefault_SetsBuiltinPreset(t *testing.T) {
	buf, run := setupPresetTest(t)

	if err := run("preset", "default", "popular"); err != nil {
		t.Fatalf("preset default failed: %v", err)
	}

	cfg, err := userconfig.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Defaults.TLDPreset != "popular" {
		t.Errorf("expected default preset 'popular', got %q", cfg.Defaults.TLDPreset)
	}
	if !strings.Contains(buf.String(), "popular") {
		t.Errorf("expected confirmation naming the preset, got %q", buf.String())
	}
}

func TestPresetDefault_SetsCustomPreset(t *testing.T) {
	_, run := setupPresetTest(t)

	if err := run("preset", "add", "nordic", "se,nu,dk"); err != nil {
		t.Fatalf("preset add failed: %v", err)
	}
	if err := run("preset", "default", "nordic"); err != nil {
		t.Fatalf("preset default failed: %v", err)
	}

	cfg, _ := userconfig.Load()
	if cfg.Defaults.TLDPreset != "nordic" {
		t.Errorf("expected default preset 'nordic', got %q", cfg.Defaults.TLDPreset)
	}
	if len(cfg.Presets["nordic"].TLDs) != 3 {
		t.Errorf("expected nordic preset to survive, got %v", cfg.Presets["nordic"].TLDs)
	}
}

func TestPresetDefault_AcceptsAll(t *testing.T) {
	_, run := setupPresetTest(t)

	if err := run("preset", "default", "all"); err != nil {
		t.Fatalf("preset default all failed: %v", err)
	}

	cfg, _ := userconfig.Load()
	if cfg.Defaults.TLDPreset != "all" {
		t.Errorf("expected default preset 'all', got %q", cfg.Defaults.TLDPreset)
	}
}

func TestPresetDefault_NormalizesName(t *testing.T) {
	_, run := setupPresetTest(t)

	if err := run("preset", "default", ".POPULAR"); err != nil {
		t.Fatalf("preset default failed: %v", err)
	}

	cfg, _ := userconfig.Load()
	if cfg.Defaults.TLDPreset != "popular" {
		t.Errorf("expected normalized name 'popular', got %q", cfg.Defaults.TLDPreset)
	}
}

func TestPresetDefault_UnknownPresetErrors(t *testing.T) {
	_, run := setupPresetTest(t)

	err := run("preset", "default", "doesnotexist")
	if err == nil {
		t.Fatal("expected error for unknown preset")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got %v", err)
	}

	cfg, _ := userconfig.Load()
	if cfg.Defaults.TLDPreset != "" {
		t.Errorf("expected no default to be written, got %q", cfg.Defaults.TLDPreset)
	}
}

func TestPresetDefault_ShowsCurrent(t *testing.T) {
	buf, run := setupPresetTest(t)

	if err := run("preset", "default"); err != nil {
		t.Fatalf("preset default failed: %v", err)
	}
	if !strings.Contains(buf.String(), "No default preset") {
		t.Errorf("expected 'no default' message, got %q", buf.String())
	}

	buf.Reset()
	if err := run("preset", "default", "tech"); err != nil {
		t.Fatalf("preset default tech failed: %v", err)
	}

	buf.Reset()
	if err := run("preset", "default"); err != nil {
		t.Fatalf("preset default failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Default preset: tech") {
		t.Errorf("expected current default in output, got %q", buf.String())
	}
}

func TestPresetDefault_Clear(t *testing.T) {
	_, run := setupPresetTest(t)

	if err := run("preset", "default", "tech"); err != nil {
		t.Fatalf("preset default failed: %v", err)
	}
	if err := run("preset", "default", "--clear"); err != nil {
		t.Fatalf("preset default --clear failed: %v", err)
	}

	cfg, _ := userconfig.Load()
	if cfg.Defaults.TLDPreset != "" {
		t.Errorf("expected default to be cleared, got %q", cfg.Defaults.TLDPreset)
	}
}

func TestPresetDefault_ClearWithNameErrors(t *testing.T) {
	_, run := setupPresetTest(t)

	if err := run("preset", "default", "--clear", "tech"); err == nil {
		t.Fatal("expected error when combining --clear with a name")
	}
}

func TestPresetRemove_ClearsDanglingDefault(t *testing.T) {
	buf, run := setupPresetTest(t)

	if err := run("preset", "add", "nordic", "se,nu"); err != nil {
		t.Fatalf("preset add failed: %v", err)
	}
	if err := run("preset", "default", "nordic"); err != nil {
		t.Fatalf("preset default failed: %v", err)
	}

	buf.Reset()
	if err := run("preset", "remove", "nordic"); err != nil {
		t.Fatalf("preset remove failed: %v", err)
	}

	cfg, _ := userconfig.Load()
	if cfg.Defaults.TLDPreset != "" {
		t.Errorf("expected default to be cleared with the preset, got %q", cfg.Defaults.TLDPreset)
	}
	if !strings.Contains(buf.String(), "Cleared default preset") {
		t.Errorf("expected notice about the cleared default, got %q", buf.String())
	}
}

func TestPresetRemove_KeepsUnrelatedDefault(t *testing.T) {
	_, run := setupPresetTest(t)

	if err := run("preset", "add", "nordic", "se,nu"); err != nil {
		t.Fatalf("preset add failed: %v", err)
	}
	if err := run("preset", "default", "tech"); err != nil {
		t.Fatalf("preset default failed: %v", err)
	}
	if err := run("preset", "remove", "nordic"); err != nil {
		t.Fatalf("preset remove failed: %v", err)
	}

	cfg, _ := userconfig.Load()
	if cfg.Defaults.TLDPreset != "tech" {
		t.Errorf("expected unrelated default to survive, got %q", cfg.Defaults.TLDPreset)
	}
}

func TestPresetList_MarksDefault(t *testing.T) {
	buf, run := setupPresetTest(t)

	if err := run("preset", "default", "tech"); err != nil {
		t.Fatalf("preset default failed: %v", err)
	}

	buf.Reset()
	if err := run("preset", "list"); err != nil {
		t.Fatalf("preset list failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "tech (default)") {
		t.Errorf("expected the default preset to be marked in the listing, got:\n%s", out)
	}
	if !strings.Contains(out, "Default preset: tech") {
		t.Errorf("expected default preset footer, got:\n%s", out)
	}
}

func TestConfigPath_PrintsPath(t *testing.T) {
	buf, run := setupPresetTest(t)

	if err := run("config", "path"); err != nil {
		t.Fatalf("config path failed: %v", err)
	}

	want, _ := userconfig.ConfigPath()
	if strings.TrimSpace(buf.String()) != want {
		t.Errorf("expected %q, got %q", want, buf.String())
	}
}

func TestConfigShow_NoFile(t *testing.T) {
	buf, run := setupPresetTest(t)

	if err := run("config", "show"); err != nil {
		t.Fatalf("config show failed: %v", err)
	}
	if !strings.Contains(buf.String(), "No config file yet") {
		t.Errorf("expected 'no config file' message, got %q", buf.String())
	}
}

func TestConfigShow_ListsSetDefaults(t *testing.T) {
	buf, run := setupPresetTest(t)

	path, _ := userconfig.ConfigPath()
	content := `
[defaults]
tlds = ["com", "se"]
max_domain_length = 12
only_available = true

[presets.nordic]
tlds = ["se", "nu"]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := run("config", "show"); err != nil {
		t.Fatalf("config show failed: %v", err)
	}

	out := buf.String()
	for _, want := range []string{"tlds", "com, se", "max_domain_length", "12", "only_available", "Custom presets: 1"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "show_stats") {
		t.Errorf("expected unset defaults to be hidden, got:\n%s", out)
	}
}

func TestConfigInit_WritesTemplate(t *testing.T) {
	_, run := setupPresetTest(t)

	if err := run("config", "init"); err != nil {
		t.Fatalf("config init failed: %v", err)
	}

	path, _ := userconfig.ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected template file: %v", err)
	}
	if !strings.Contains(string(data), "[defaults]") {
		t.Errorf("expected a [defaults] section, got:\n%s", data)
	}

	cfg, err := userconfig.Load()
	if err != nil {
		t.Fatalf("template is not valid TOML: %v", err)
	}
	if !reflect.DeepEqual(cfg.Defaults, userconfig.Defaults{}) {
		t.Errorf("expected template to set nothing, got %+v", cfg.Defaults)
	}
}

func TestConfigInit_RefusesToClobber(t *testing.T) {
	_, run := setupPresetTest(t)

	if err := run("preset", "add", "nordic", "se,nu"); err != nil {
		t.Fatalf("preset add failed: %v", err)
	}

	err := run("config", "init")
	if err == nil {
		t.Fatal("expected error when config file already exists")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("expected error to mention --force, got %v", err)
	}

	cfg, _ := userconfig.Load()
	if _, ok := cfg.Presets["nordic"]; !ok {
		t.Error("expected existing config to be left intact")
	}
}

func TestConfigInit_ForceOverwrites(t *testing.T) {
	_, run := setupPresetTest(t)

	if err := run("preset", "add", "nordic", "se,nu"); err != nil {
		t.Fatalf("preset add failed: %v", err)
	}
	if err := run("config", "init", "--force"); err != nil {
		t.Fatalf("config init --force failed: %v", err)
	}

	cfg, _ := userconfig.Load()
	if len(cfg.Presets) != 0 {
		t.Errorf("expected file to be replaced by the template, got %v", cfg.Presets)
	}
}

func TestConfigShow_ListsForSaleDefaults(t *testing.T) {
	buf, run := setupPresetTest(t)

	path, _ := userconfig.ConfigPath()
	content := `
[defaults]
for_sale = true
only_for_sale = true
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := run("config", "show"); err != nil {
		t.Fatalf("config show failed: %v", err)
	}

	out := buf.String()
	for _, want := range []string{"for_sale", "only_for_sale"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got:\n%s", want, out)
		}
	}
}
