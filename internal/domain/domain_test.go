package domain

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/openrdap/rdap"
)

func TestIsValidDomainOrKeyword(t *testing.T) {
	Config.MaxDomainLength = 63
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
		result := isValidDomainOrKeyword(test.input)
		if result != test.expected {
			t.Errorf("isValidDomainOrKeyword(%s) = %v; expected %v", test.input, result, test.expected)
		}
	}
}

func TestMaxDomainLength(t *testing.T) {
	Config.MaxDomainLength = 10
	tests := []struct {
		input    string
		expected bool
	}{
		{"example", true},
		{"asdfghhasd", true},
		{"asdfghhasdj", false},
		{"a-very-long-domain-name-that-is-invalid-because-its-over-63-characters-long", false},
	}

	for _, test := range tests {
		result := isValidDomainOrKeyword(test.input)
		if result != test.expected {
			t.Errorf("isValidDomainOrKeyword(%s) = %v; expected %v", test.input, result, test.expected)
		}
	}

	Config.MaxDomainLength = 63
}

func TestRemoveDuplicates(t *testing.T) {
	input := []string{"apple", "banana", "apple", "cherry", "banana"}
	expected := []string{"apple", "banana", "cherry"}
	result := removeDuplicates(input)

	if len(result) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(result))
	}

	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("Expected %s at index %d, got %s", expected[i], i, result[i])
		}
	}
}

func TestValidateKeywords(t *testing.T) {
	Config = ConfigOptions{MaxDomainLength: 63}
	input := []string{"google.com", "example", "example.com", "test.org", "google.co.uk"}
	expected := []string{"google", "example", "test"}

	result := validateKeywords(input)

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
		found := slices.Contains(Config.TLDs, tld)
		if !found {
			t.Errorf("Expected TLD %s not added to config. Instead got %s", tld, Config.TLDs)
		}
	}
}

func TestCheckAvailability_InvalidDomain(t *testing.T) {
	Config.MaxDomainLength = 63
	s := ResolverService{
		config:     &Config,
		rdapClient: &rdap.Client{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := s.CheckDomain(ctx, "@@@invalid###.com")
	if err == nil {
		t.Errorf("Expected error for invalid domain")
	}
}

func TestCheckAvailability_Timeout(t *testing.T) {
	Config.MaxDomainLength = 63
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond) // force timeout
	defer cancel()

	s := ResolverService{
		config:     &Config,
		rdapClient: &rdap.Client{},
	}

	_, err := s.CheckDomain(ctx, "example.com")
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context deadline exceeded, got %v", err)
	}
}

func TestGenerateDomainPermutations(t *testing.T) {
	tests := []struct {
		input    []string
		tlds     []string
		prefixes []string
		suffixes []string
		expected []string
	}{
		{
			input:    []string{"example", "test"},
			tlds:     []string{},
			prefixes: []string{},
			suffixes: []string{},
			expected: []string{"example.com", "test.com"},
		},
		{
			input:    []string{"example", "test"},
			tlds:     []string{"com", "org"},
			prefixes: []string{},
			suffixes: []string{},
			expected: []string{"example.com", "example.org", "test.com", "test.org"},
		},
		{
			input:    []string{"test"},
			tlds:     []string{},
			prefixes: []string{"use"},
			suffixes: []string{"ly", "now"},
			expected: []string{
				"usetestly.com",
				"usetestnow.com",
				"testly.com",
				"testnow.com",
				"test.com",
				"usetest.com",
			},
		},
		{
			input:    []string{"test"},
			tlds:     []string{"com", "com", "com"},
			prefixes: []string{"use", "use"},
			suffixes: []string{"ly", "ly"},
			expected: []string{
				"test.com",
				"usetest.com",
				"usetestly.com",
				"testly.com",
			},
		},
	}

	Config.MaxDomainLength = 63
	for _, test := range tests {
		Config.TLDs = test.tlds
		Config.Prefixes = test.prefixes
		Config.Suffixes = test.suffixes
		result := generateDomainPermutations(test.input)
		if len(result) != len(test.expected) {
			t.Errorf("Expected %d permutations, got %d", len(test.expected), len(result))
		}
		for _, perm := range test.expected {
			if !slices.Contains(result, perm) {
				t.Errorf("Expected permutation %s not found in result", perm)
			}
		}
	}
}
