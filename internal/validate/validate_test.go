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
	input := []string{"google.com", "example", "example.com", "test.org", "google.co.uk"}
	expected := []string{"google", "example", "test"}

	validatedKeywords := validate.ValidateKeywords(input)
	result := validatedKeywords.Keywords
	tlds := validatedKeywords.NewTlds

	if len(result) != len(expected) {
		t.Errorf("Expected %d keywords, got %d", len(expected), len(result))
	}

	for _, keyword := range expected {
		found := slices.Contains(result, keyword)
		if !found {
			t.Errorf("Expected keyword %s not found", keyword)
		}
	}

	expectedTLDs := []string{"com", "org", "co.uk"}
	for _, tld := range expectedTLDs {
		found := slices.Contains(tlds, tld)
		if !found {
			t.Errorf("Expected TLD %s not added to config. Instead got %s", tld, tlds)
		}
	}
}
