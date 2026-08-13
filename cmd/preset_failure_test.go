package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPresetDefault_ClearWithNoDefaultSet(t *testing.T) {
	buf, run := setupPresetTest(t)

	if err := run("preset", "default", "--clear"); err != nil {
		t.Fatalf("preset default --clear failed: %v", err)
	}
	if !strings.Contains(buf.String(), "No default preset is set") {
		t.Errorf("expected the no-op message, got %q", buf.String())
	}
}

func TestPresetDefault_EmptyNameErrors(t *testing.T) {
	_, run := setupPresetTest(t)

	err := run("preset", "default", ".")
	if err == nil {
		t.Fatal("expected an error for an empty preset name")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("expected an 'empty name' error, got %v", err)
	}
}

func TestPresetDefault_SurfacesAnUnreadableConfig(t *testing.T) {
	_, run := setupPresetTest(t)

	path := filepath.Join(t.TempDir(), "broken.toml")
	if err := os.WriteFile(path, []byte("[presets.nordic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TLDX_CONFIG", path)

	if err := run("preset", "default", "popular"); err == nil {
		t.Fatal("expected an error for a malformed config file")
	}
}

func TestPresetDefault_SurfacesASaveFailure(t *testing.T) {
	_, run := setupPresetTest(t)
	readOnlyConfig(t, "")

	err := run("preset", "default", "popular")
	if err == nil {
		t.Fatal("expected an error when the config file cannot be written")
	}
	if !strings.Contains(err.Error(), "write") {
		t.Errorf("expected a write error, got %v", err)
	}
}

func TestPresetDefault_ClearSurfacesASaveFailure(t *testing.T) {
	_, run := setupPresetTest(t)
	readOnlyConfig(t, "[defaults]\ntld_preset = \"popular\"\n")

	if err := run("preset", "default", "--clear"); err == nil {
		t.Fatal("expected an error when the config file cannot be written")
	}
}

func TestPresetAdd_SurfacesASaveFailure(t *testing.T) {
	_, run := setupPresetTest(t)
	readOnlyConfig(t, "")

	if err := run("preset", "add", "nordic", "se,nu"); err == nil {
		t.Fatal("expected an error when the config file cannot be written")
	}
}

func TestPresetRemove_SurfacesASaveFailure(t *testing.T) {
	_, run := setupPresetTest(t)
	readOnlyConfig(t, "[presets.nordic]\ntlds = [\"se\", \"nu\"]\n")

	if err := run("preset", "remove", "nordic"); err == nil {
		t.Fatal("expected an error when the config file cannot be written")
	}
}

func TestPresetList_MarksAllAsTheDefault(t *testing.T) {
	buf, run := setupPresetTest(t)

	if err := run("preset", "default", "all"); err != nil {
		t.Fatalf("preset default all failed: %v", err)
	}

	buf.Reset()
	if err := run("preset", "list"); err != nil {
		t.Fatalf("preset list failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "all (default)") {
		t.Errorf("expected 'all' to be marked as the default, got:\n%s", out)
	}
	if !strings.Contains(out, "Default preset: all") {
		t.Errorf("expected the default preset footer, got:\n%s", out)
	}
}
