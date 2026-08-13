package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brandonyoungdev/tldx/internal/userconfig"
)

// withoutConfigDir makes userconfig.ConfigPath fail by removing every variable
// os.UserConfigDir consults.
func withoutConfigDir(t *testing.T) {
	t.Helper()
	t.Setenv("TLDX_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
}

// readOnlyConfig writes contents to the config file and makes it unwritable, so
// loading succeeds but saving fails.
func readOnlyConfig(t *testing.T, contents string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(contents), 0o400); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TLDX_CONFIG", path)

	// root ignores the permission bits, so there is nothing to assert on.
	if f, err := os.OpenFile(path, os.O_WRONLY, 0o400); err == nil {
		f.Close()
		t.Skip("running as root: a read-only config file is still writable")
	}
}

func TestConfigPath_ErrorsWithoutAConfigDir(t *testing.T) {
	_, run := setupPresetTest(t)
	withoutConfigDir(t)

	if err := run("config", "path"); err == nil {
		t.Fatal("expected an error when the config directory cannot be determined")
	}
}

func TestConfigShow_ErrorsWithoutAConfigDir(t *testing.T) {
	_, run := setupPresetTest(t)
	withoutConfigDir(t)

	if err := run("config", "show"); err == nil {
		t.Fatal("expected an error when the config directory cannot be determined")
	}
}

func TestConfigInit_ErrorsWithoutAConfigDir(t *testing.T) {
	_, run := setupPresetTest(t)
	withoutConfigDir(t)

	if err := run("config", "init"); err == nil {
		t.Fatal("expected an error when the config directory cannot be determined")
	}
}

func TestConfigShow_SurfacesAnUnreadableConfig(t *testing.T) {
	_, run := setupPresetTest(t)

	path := filepath.Join(t.TempDir(), "broken.toml")
	if err := os.WriteFile(path, []byte("[defaults\ntlds = "), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TLDX_CONFIG", path)

	err := run("config", "show")
	if err == nil {
		t.Fatal("expected an error for a malformed config file")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("expected a parse error, got %v", err)
	}
}

func TestConfigShow_ReportsNoDefaults(t *testing.T) {
	buf, run := setupPresetTest(t)

	if err := run("preset", "add", "nordic", "se,nu"); err != nil {
		t.Fatalf("preset add failed: %v", err)
	}

	buf.Reset()
	if err := run("config", "show"); err != nil {
		t.Fatalf("config show failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Defaults: none set") {
		t.Errorf("expected 'none set' for a config with only presets, got:\n%s", out)
	}
	if !strings.Contains(out, "Custom presets: 1") {
		t.Errorf("expected the preset count, got:\n%s", out)
	}
}

func TestConfigShow_ListsEveryKindOfDefault(t *testing.T) {
	buf, run := setupPresetTest(t)

	path, _ := userconfig.ConfigPath()
	content := `
[defaults]
tld_preset = "popular"
prefixes = ["get", "my"]
suffixes = ["ly"]
format = "json"
limit = 5
show_stats = true
no_color = true
verbose = true
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := run("config", "show"); err != nil {
		t.Fatalf("config show failed: %v", err)
	}

	out := buf.String()
	want := []string{
		"tld_preset", "popular",
		"prefixes", "get, my",
		"suffixes", "ly",
		"format", "json",
		"limit", "5",
		"show_stats", "no_color", "verbose",
	}
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("expected %q in output, got:\n%s", w, out)
		}
	}
}

func TestConfigInit_ErrorsWhenTheParentIsAFile(t *testing.T) {
	_, run := setupPresetTest(t)

	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TLDX_CONFIG", filepath.Join(blocker, "config.toml"))

	err := run("config", "init")
	if err == nil {
		t.Fatal("expected an error when the config directory cannot be created")
	}
	if !strings.Contains(err.Error(), "create config dir") {
		t.Errorf("expected a 'create config dir' error, got %v", err)
	}
}

func TestConfigInit_ErrorsWhenThePathIsADirectory(t *testing.T) {
	_, run := setupPresetTest(t)

	dir := filepath.Join(t.TempDir(), "config.toml")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TLDX_CONFIG", dir)

	// Without --force the existing directory is reported as an existing file.
	err := run("config", "init", "--force")
	if err == nil {
		t.Fatal("expected an error when the config path is not writable")
	}
	if !strings.Contains(err.Error(), "write") {
		t.Errorf("expected a write error, got %v", err)
	}
}
