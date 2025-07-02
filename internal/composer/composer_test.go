package composer_test

import (
	"slices"
	"testing"

	"github.com/brandonyoungdev/tldx/internal/composer"
	"github.com/brandonyoungdev/tldx/internal/config"
)

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

	for _, test := range tests {

		app := config.NewTldxContext()
		s := composer.NewComposerService(app)
		app.Config.TLDs = test.tlds
		app.Config.Prefixes = test.prefixes
		app.Config.Suffixes = test.suffixes
		result, warning := s.GenerateDomainPermutations(test.input)
		if len(warning) != 0 {
			t.Errorf("Unexpected warnings: %v", warning)
		}
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
