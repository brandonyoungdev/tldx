package cmd_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brandonyoungdev/tldx/cmd"
	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/userconfig"
)

func setupPresetTest(t *testing.T) (*bytes.Buffer, func(args ...string) error) {
	t.Helper()

	tmp := t.TempDir()
	t.Setenv("TLDX_CONFIG", filepath.Join(tmp, "presets.toml"))

	app := config.NewTldxContext()
	root := cmd.NewRootCmd(app)

	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)

	run := func(args ...string) error {
		root.SetArgs(args)
		return root.ExecuteContext(context.Background())
	}
	return buf, run
}

func TestPresetAdd_CreatesPreset(t *testing.T) {
	_, run := setupPresetTest(t)

	if err := run("preset", "add", "myteam", "com,io,ai"); err != nil {
		t.Fatalf("preset add failed: %v", err)
	}

	cfg, err := userconfig.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	entry, ok := cfg.Presets["myteam"]
	if !ok {
		t.Fatal("expected preset 'myteam' to exist after add")
	}
	if len(entry.TLDs) != 3 {
		t.Errorf("expected 3 TLDs, got %v", entry.TLDs)
	}
}

func TestPresetAdd_SpaceSeparated(t *testing.T) {
	_, run := setupPresetTest(t)

	if err := run("preset", "add", "spacey", "com", "io", "dev"); err != nil {
		t.Fatalf("preset add with separate args failed: %v", err)
	}

	cfg, _ := userconfig.Load()
	entry := cfg.Presets["spacey"]
	if len(entry.TLDs) != 3 {
		t.Errorf("expected 3 TLDs from separate-arg input, got %v", entry.TLDs)
	}
}

func TestPresetAdd_StripsLeadingDots(t *testing.T) {
	_, run := setupPresetTest(t)

	if err := run("preset", "add", "dotted", ".com,.io"); err != nil {
		t.Fatalf("preset add failed: %v", err)
	}

	cfg, _ := userconfig.Load()
	for _, tld := range cfg.Presets["dotted"].TLDs {
		if strings.HasPrefix(tld, ".") {
			t.Errorf("TLD should not have leading dot: %s", tld)
		}
	}
}

func TestPresetAdd_ReplacesExistingPreset(t *testing.T) {
	_, run := setupPresetTest(t)

	_ = run("preset", "add", "myteam", "com,io")
	_ = run("preset", "add", "myteam", "net,org,co")

	cfg, _ := userconfig.Load()
	entry := cfg.Presets["myteam"]
	if len(entry.TLDs) != 3 {
		t.Errorf("expected 3 TLDs after replace, got %v", entry.TLDs)
	}
}

func TestPresetAdd_InvalidTLD_ReturnsError(t *testing.T) {
	_, run := setupPresetTest(t)

	err := run("preset", "add", "bad", "comssdfa")
	if err == nil {
		t.Fatal("expected error for invalid TLD 'comssdfa'")
	}
	if !strings.Contains(err.Error(), "comssdfa") {
		t.Errorf("expected error to mention the invalid TLD, got: %v", err)
	}
}

func TestPresetAdd_MixedValidInvalid_ReturnsError(t *testing.T) {
	_, run := setupPresetTest(t)

	err := run("preset", "add", "mixed", "com", "fakeinvalidtld999")
	if err == nil {
		t.Fatal("expected error when any TLD is invalid")
	}
}

func TestPresetAdd_MultiLabelTLD_Accepted(t *testing.T) {
	_, run := setupPresetTest(t)

	if err := run("preset", "add", "brit", "co.uk"); err != nil {
		t.Fatalf("co.uk should be accepted as a valid TLD, got: %v", err)
	}
	cfg, _ := userconfig.Load()
	if _, ok := cfg.Presets["brit"]; !ok {
		t.Error("expected 'brit' preset to be saved")
	}
}

func TestPresetAdd_EmptyTLDs_ReturnsError(t *testing.T) {
	_, run := setupPresetTest(t)

	err := run("preset", "add", "empty", ",,,")
	if err == nil {
		t.Fatal("expected error for empty TLD list")
	}
}

func TestPresetRemove_RemovesUserPreset(t *testing.T) {
	_, run := setupPresetTest(t)

	_ = run("preset", "add", "todelete", "com")
	if err := run("preset", "remove", "todelete"); err != nil {
		t.Fatalf("preset remove failed: %v", err)
	}

	cfg, _ := userconfig.Load()
	if _, ok := cfg.Presets["todelete"]; ok {
		t.Error("expected preset to be removed")
	}
}

func TestPresetRemove_BuiltinPreset_ReturnsError(t *testing.T) {
	_, run := setupPresetTest(t)

	err := run("preset", "remove", "popular")
	if err == nil {
		t.Fatal("expected error when removing built-in preset")
	}
}

func TestPresetRemove_NotFound_ReturnsError(t *testing.T) {
	_, run := setupPresetTest(t)

	err := run("preset", "remove", "doesnotexist")
	if err == nil {
		t.Fatal("expected error when removing non-existent preset")
	}
}

func TestPresetList_ShowsBuiltinsAndCustom(t *testing.T) {
	buf, run := setupPresetTest(t)

	_ = run("preset", "add", "custompreset", "com,io")

	buf.Reset()
	if err := run("preset", "list"); err != nil {
		t.Fatalf("preset list failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "popular") {
		t.Error("expected built-in preset 'popular' in output")
	}
	if !strings.Contains(out, "custompreset") {
		t.Error("expected custom preset 'custompreset' in output")
	}
	if !strings.Contains(out, "*") {
		t.Error("expected '*' annotation for custom preset")
	}
	if !strings.Contains(out, "presets.toml") {
		t.Error("expected config file path in output")
	}
}

func TestPresetAdd_EmptyName_ReturnsError(t *testing.T) {
	_, run := setupPresetTest(t)

	err := run("preset", "add", "", "com")
	if err == nil {
		t.Fatal("expected error for empty preset name")
	}
}

func TestPresetAdd_SpaceOnlyPart_Filtered(t *testing.T) {
	_, run := setupPresetTest(t)

	// "com, ,io" has a whitespace-only part between the commas; it should be
	// silently filtered and the preset saved with ["com", "io"].
	if err := run("preset", "add", "filtered", "com, ,io"); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	cfg, _ := userconfig.Load()
	entry := cfg.Presets["filtered"]
	if len(entry.TLDs) != 2 {
		t.Errorf("expected 2 TLDs after filtering empty part, got %v", entry.TLDs)
	}
}

func TestPresetAdd_MalformedConfig_ReturnsError(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "presets.toml")
	t.Setenv("TLDX_CONFIG", path)
	if err := os.WriteFile(path, []byte("[[[invalid toml"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := config.NewTldxContext()
	root := cmd.NewRootCmd(app)
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"preset", "add", "test", "com"})
	if err := root.ExecuteContext(context.Background()); err == nil {
		t.Fatal("expected error for malformed config in preset add")
	}
}

func TestPresetRemove_MalformedConfig_ReturnsError(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "presets.toml")
	t.Setenv("TLDX_CONFIG", path)
	if err := os.WriteFile(path, []byte("[[[invalid toml"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := config.NewTldxContext()
	root := cmd.NewRootCmd(app)
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"preset", "remove", "test"})
	if err := root.ExecuteContext(context.Background()); err == nil {
		t.Fatal("expected error for malformed config in preset remove")
	}
}

func TestPresetList_MalformedConfig_FallsBackToBuiltins(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "presets.toml")
	t.Setenv("TLDX_CONFIG", path)
	if err := os.WriteFile(path, []byte("[[[invalid toml"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := config.NewTldxContext()
	root := cmd.NewRootCmd(app)
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"preset", "list"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("preset list should succeed even with malformed config, got: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "popular") {
		t.Error("expected built-in preset 'popular' in fallback output")
	}
}

func TestPresetList_EmptyUserConfig_ShowsBuiltins(t *testing.T) {
	buf, run := setupPresetTest(t)

	if err := run("preset", "list"); err != nil {
		t.Fatalf("preset list failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "popular") {
		t.Error("expected built-in preset 'popular' in output")
	}
	if !strings.Contains(out, "tech") {
		t.Error("expected built-in preset 'tech' in output")
	}
}
