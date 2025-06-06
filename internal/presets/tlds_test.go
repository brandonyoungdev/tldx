package presets_test

import (
	"slices"
	"testing"

	"github.com/brandonyoungdev/tldx/internal/presets"
)

func TestGetAllTLDs(t *testing.T) {
	all := presets.GetAllTLDs()

	if len(all) == 0 {
		t.Fatal("GetAllTLDs returned empty slice, expected some TLDs")
	}

	// Check sorted order
	if !slices.IsSorted(all) {
		t.Error("GetAllTLDs: returned slice is not sorted")
	}

	// Check deduplication: no adjacent duplicates
	for i := 1; i < len(all); i++ {
		if all[i] == all[i-1] {
			t.Errorf("GetAllTLDs: found duplicate entry %q at positions %d and %d", all[i], i-1, i)
		}
	}

	// Spot-check known TLDs are present
	expected := []string{"com", "net", "org", "io", "me", "dev", "app", "ai"}
	for _, want := range expected {
		i := slices.Index(all, want)
		if i == -1 {
			t.Errorf("GetAllTLDs: expected to find %q but did not", want)
		}
	}
}
