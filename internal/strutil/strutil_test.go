package strutil_test

import (
	"reflect"
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

func TestAllToLowerCase(t *testing.T) {
	tests := []struct {
		input    []string
		expected []string
	}{
		{[]string{"Hello", "WORLD"}, []string{"hello", "world"}},
		{[]string{"already", "lower"}, []string{"already", "lower"}},
		{[]string{"MiXeD", "CaSe"}, []string{"mixed", "case"}},
		{[]string{}, []string{}},
		{[]string{"GO", "LANG"}, []string{"go", "lang"}},
	}

	for _, tt := range tests {
		got := strutil.AllToLowerCase(tt.input)
		if !reflect.DeepEqual(got, tt.expected) {
			t.Errorf("AllToLowerCase(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestFilterByMaxLength(t *testing.T) {
	tests := []struct {
		input     []string
		maxLength int
		expected  []string
	}{
		{[]string{"a", "bb", "ccc", "dddd"}, 2, []string{"a", "bb"}},
		{[]string{"a", "bb"}, 5, []string{"a", "bb"}},
		{[]string{"abc", "def"}, 3, []string{"abc", "def"}},
		{[]string{"abc", "de"}, 2, []string{"de"}},
		{[]string{}, 5, nil},
		{[]string{"toolong"}, 3, nil},
	}

	for _, tt := range tests {
		got := strutil.FilterByMaxLength(tt.input, tt.maxLength)
		if !reflect.DeepEqual(got, tt.expected) {
			t.Errorf("FilterByMaxLength(%v, %d) = %v, want %v", tt.input, tt.maxLength, got, tt.expected)
		}
	}
}
