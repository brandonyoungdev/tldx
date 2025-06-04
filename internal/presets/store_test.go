package presets

import (
	"reflect"
	"testing"
)

type TLDPreset []string

func TestPresetStore_Get(t *testing.T) {
	builtin := map[string]TLDPreset{
		"popular": {"com", "net", "org"},
		"tech":    {"dev", "io"},
	}
	ps := NewTypedStore("TLD", builtin)

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
	ps := NewTypedStore("TLD", builtin)

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
	ps := NewTypedStore("TLD", builtin)
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
	ps := NewTypedStore("TLD", map[string]TLDPreset{})
	if ps.Kind != "TLD" {
		t.Errorf("Expected kind to be 'TLD', got %s", ps.Kind)
	}
}
