package presets_test

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/brandonyoungdev/tldx/internal/presets"
)

type TLDPreset []string

func TestPresetStore_Get(t *testing.T) {
	builtin := map[string]TLDPreset{
		"popular": {"com", "net", "org"},
		"tech":    {"dev", "io"},
	}
	ps := presets.NewTypedStore("TLD", builtin)

	t.Run("gets builtin preset", func(t *testing.T) {
		preset, ok := ps.Get("popular")
		if !ok {
			t.Fatalf("Expected preset 'popular' to exist")
		}
		expected := TLDPreset{"com", "net", "org"}
		if !reflect.DeepEqual(preset, expected) {
			t.Errorf("Expected %v, got %v", expected, preset)
		}
	})

	t.Run("returns false for missing preset", func(t *testing.T) {
		_, ok := ps.Get("nonexistent")
		if ok {
			t.Errorf("Expected 'nonexistent' to be missing")
		}
	})
}

func TestPresetStore_Override(t *testing.T) {
	builtin := map[string]TLDPreset{
		"popular": {"com", "net", "org"},
	}
	ps := presets.NewTypedStore("TLD", builtin)

	override := TLDPreset{"xyz"}
	ps.Override("popular", override)

	got, ok := ps.Get("popular")
	if !ok {
		t.Fatalf("Expected override to be found")
	}
	if !reflect.DeepEqual(got, override) {
		t.Errorf("Expected override %v, got %v", override, got)
	}
}

func TestPresetStore_All(t *testing.T) {
	builtin := map[string]TLDPreset{
		"a": {"com"},
		"b": {"net"},
	}
	ps := presets.NewTypedStore("TLD", builtin)
	ps.Override("b", TLDPreset{"overridden"})
	ps.Override("c", TLDPreset{"custom"})

	all := ps.All()
	expected := map[string]TLDPreset{
		"a": {"com"},
		"b": {"overridden"},
		"c": {"custom"},
	}

	if !reflect.DeepEqual(all, expected) {
		t.Errorf("Expected all presets to be %v, got %v", expected, all)
	}
}

func TestPresetStore_Kind(t *testing.T) {
	ps := presets.NewTypedStore("TLD", map[string]TLDPreset{})
	if ps.Kind != "TLD" {
		t.Errorf("Expected kind to be 'TLD', got %s", ps.Kind)
	}
}

func TestShowAllPresets_OutputContainsNames(t *testing.T) {
	builtin := map[string]TLDPreset{
		"popular": {"com", "net", "org"},
		"tech":    {"dev", "io"},
	}
	ps := presets.NewTypedStore("TLD", builtin)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	presets.ShowAllPresets(ps, func(v TLDPreset) string {
		return strings.Join(v, " ")
	})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "popular") {
		t.Errorf("Expected output to contain 'popular', got: %s", output)
	}
	if !strings.Contains(output, "tech") {
		t.Errorf("Expected output to contain 'tech', got: %s", output)
	}
	if !strings.Contains(output, "TLD") {
		t.Errorf("Expected output to contain kind 'TLD', got: %s", output)
	}
}

func TestShowAllPresets_AlphabeticOrder(t *testing.T) {
	builtin := map[string]TLDPreset{
		"zebra": {"z"},
		"apple": {"a"},
		"mango": {"m"},
	}
	ps := presets.NewTypedStore("TLD", builtin)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	presets.ShowAllPresets(ps, func(v TLDPreset) string {
		return strings.Join(v, " ")
	})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	appleIdx := strings.Index(out, "apple")
	mangoIdx := strings.Index(out, "mango")
	zebraIdx := strings.Index(out, "zebra")

	if appleIdx > mangoIdx || mangoIdx > zebraIdx {
		t.Errorf("Expected alphabetical order apple < mango < zebra, got positions %d %d %d", appleIdx, mangoIdx, zebraIdx)
	}
}

func TestShowAllPresets_LongTLDListWraps(t *testing.T) {
	longTLDs := make(TLDPreset, 30)
	for i := range longTLDs {
		longTLDs[i] = fmt.Sprintf("tld%02d", i)
	}
	builtin := map[string]TLDPreset{"longlabel": longTLDs}
	ps := presets.NewTypedStore("TLD", builtin)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	presets.ShowAllPresets(ps, func(v TLDPreset) string {
		return strings.Join(v, " ")
	})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	// Should contain multiple lines for the long preset
	lineCount := strings.Count(out, "\n")
	if lineCount < 3 {
		t.Errorf("Expected multiple lines for wrapped output, got %d lines", lineCount)
	}
}
