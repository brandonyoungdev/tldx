package validate_test

import (
	"slices"
	"testing"

	"github.com/brandonyoungdev/tldx/internal/validate"
)

func TestIsValidDomainOrKeyword(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"example", true},
		{"ex-ample", true},
		{"exa_mple", false},
		{"-example", false},
		{"example-", false},
		{"a-very-long-domain-name-that-is-invalid-because-its-over-63-characters-long", false},
		{"exa.mple", true},
	}

	for _, test := range tests {
		result := validate.IsValidDomainOrKeyword(test.input)
		if result != test.expected {
			t.Errorf("isValidDomainOrKeyword(%s) = %v; expected %v", test.input, result, test.expected)
		}
	}
}

func TestValidateKeywords(t *testing.T) {
	input := []string{"Google.com", "example", "example.com", "example", "google.co.uk"}
	expected := []string{"google.com", "example", "example.com", "google.co.uk"}

	if result := validate.ValidateKeywords(input); !slices.Equal(result, expected) {
		t.Errorf("ValidateKeywords(%v) = %v; expected %v", input, result, expected)
	}
}

func TestValidateKeywords_InvalidKeywordSkipped(t *testing.T) {
	// "@invalid" fails IsValidDomainOrKeyword → triggers the continue branch
	input := []string{"valid", "@invalid", "also-valid"}
	result := validate.ValidateKeywords(input)
	if slices.Contains(result, "@invalid") {
		t.Error("expected @invalid to be skipped")
	}
	if !slices.Contains(result, "valid") || !slices.Contains(result, "also-valid") {
		t.Error("expected valid keywords to be present")
	}
}
