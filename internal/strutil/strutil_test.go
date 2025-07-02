package strutil_test

import (
	"testing"

	"github.com/brandonyoungdev/tldx/internal/strutil"
)

func TestRemoveDuplicates(t *testing.T) {
	input := []string{"apple", "banana", "apple", "cherry", "banana"}
	expected := []string{"apple", "banana", "cherry"}
	result := strutil.RemoveDuplicates(input)

	if len(result) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(result))
	}

	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("Expected %s at index %d, got %s", expected[i], i, result[i])
		}
	}
}
