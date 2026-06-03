package userconfig_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/brandonyoungdev/tldx/internal/userconfig"
)

func withTempConfigPath(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "presets.toml")
	t.Setenv("TLDX_CONFIG", path)
	return path
}

func TestConfigPath_ReturnsPath(t *testing.T) {
	path := withTempConfigPath(t)

	got, err := userconfig.ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error: %v", err)
	}
	if got != path {
		t.Errorf("expected %s, got %s", path, got)
	}
	if filepath.Base(got) != "presets.toml" {
		t.Errorf("expected filename presets.toml, got %s", filepath.Base(got))
	}
}

func TestLoad_MissingFile_ReturnsEmpty(t *testing.T) {
	withTempConfigPath(t)

	cfg, err := userconfig.Load()
	if err != nil {
		t.Fatalf("Load() with missing file should not error, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}
	if len(cfg.Presets) != 0 {
		t.Errorf("expected empty presets, got %v", cfg.Presets)
	}
}

func TestLoad_ValidTOML_ParsesPresets(t *testing.T) {
	path := withTempConfigPath(t)

	content := `
[presets.myteam]
tlds = ["com", "io", "ai"]

[presets.saas]
tlds = ["app", "dev"]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := userconfig.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	expected := map[string]userconfig.PresetEntry{
		"myteam": {TLDs: []string{"com", "io", "ai"}},
		"saas":   {TLDs: []string{"app", "dev"}},
	}
	if !reflect.DeepEqual(cfg.Presets, expected) {
		t.Errorf("expected %v, got %v", expected, cfg.Presets)
	}
}

func TestLoad_MalformedTOML_ReturnsError(t *testing.T) {
	path := withTempConfigPath(t)

	if err := os.WriteFile(path, []byte("[[[invalid toml"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := userconfig.Load()
	if err == nil {
		t.Fatal("expected error for malformed TOML, got nil")
	}
}

func TestSave_RoundTrip(t *testing.T) {
	withTempConfigPath(t)

	original := &userconfig.UserConfig{
		Presets: map[string]userconfig.PresetEntry{
			"startup": {TLDs: []string{"com", "io", "co"}},
		},
	}

	if err := userconfig.Save(original); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := userconfig.Load()
	if err != nil {
		t.Fatalf("Load() after Save() error: %v", err)
	}
	if !reflect.DeepEqual(original.Presets, loaded.Presets) {
		t.Errorf("round-trip mismatch: saved %v, loaded %v", original.Presets, loaded.Presets)
	}
}

func TestSave_CreatesParentDirs(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "nested", "deep", "presets.toml")
	t.Setenv("TLDX_CONFIG", path)

	cfg := &userconfig.UserConfig{
		Presets: map[string]userconfig.PresetEntry{
			"test": {TLDs: []string{"com"}},
		},
	}

	if err := userconfig.Save(cfg); err != nil {
		t.Fatalf("Save() should create dirs, got error: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file to be created at %s", path)
	}
}

func TestConfigPath_WithoutEnvOverride(t *testing.T) {
	t.Setenv("TLDX_CONFIG", "")

	got, err := userconfig.ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() without env override: %v", err)
	}
	if got == "" {
		t.Fatal("expected non-empty path")
	}
	if filepath.Base(got) != "presets.toml" {
		t.Errorf("expected filename presets.toml, got %s", filepath.Base(got))
	}
}

func TestLoad_FileReadError_ReturnsError(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "presets.toml")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TLDX_CONFIG", path)

	_, err := userconfig.Load()
	if err == nil {
		t.Fatal("expected error when config path is a directory")
	}
}

func TestLoad_EmptyTOML_InitializesPresets(t *testing.T) {
	path := withTempConfigPath(t)

	// Write valid TOML that has no [presets] section.
	if err := os.WriteFile(path, []byte("# no presets here\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := userconfig.Load()
	if err != nil {
		t.Fatalf("Load() on empty TOML: %v", err)
	}
	if cfg.Presets == nil {
		t.Error("expected Presets map to be initialized, got nil")
	}
}

func TestSave_MkdirAllError_ReturnsError(t *testing.T) {
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "blocked")
	if err := os.WriteFile(blocker, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(blocker, "presets.toml")
	t.Setenv("TLDX_CONFIG", path)

	cfg := &userconfig.UserConfig{Presets: map[string]userconfig.PresetEntry{
		"test": {TLDs: []string{"com"}},
	}}

	err := userconfig.Save(cfg)
	if err == nil {
		t.Fatal("expected error when parent path is a file")
	}
}

func TestSave_CreateFileError_ReturnsError(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "presets.toml")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TLDX_CONFIG", path)

	cfg := &userconfig.UserConfig{Presets: map[string]userconfig.PresetEntry{
		"test": {TLDs: []string{"com"}},
	}}

	err := userconfig.Save(cfg)
	if err == nil {
		t.Fatal("expected error when config path is a directory")
	}
}

func TestSave_OverwritesExisting(t *testing.T) {
	withTempConfigPath(t)

	first := &userconfig.UserConfig{
		Presets: map[string]userconfig.PresetEntry{
			"alpha": {TLDs: []string{"com"}},
		},
	}
	if err := userconfig.Save(first); err != nil {
		t.Fatalf("first Save() error: %v", err)
	}

	second := &userconfig.UserConfig{
		Presets: map[string]userconfig.PresetEntry{
			"beta": {TLDs: []string{"io"}},
		},
	}
	if err := userconfig.Save(second); err != nil {
		t.Fatalf("second Save() error: %v", err)
	}

	loaded, err := userconfig.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if _, ok := loaded.Presets["alpha"]; ok {
		t.Error("expected alpha preset to be gone after overwrite")
	}
	if _, ok := loaded.Presets["beta"]; !ok {
		t.Error("expected beta preset after overwrite")
	}
}
