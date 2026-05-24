package composer_test

import (
	"slices"
	"testing"

	"github.com/brandonyoungdev/tldx/internal/composer"
	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/resolver"
	"github.com/stretchr/testify/assert"
)

func specDomains(specs []resolver.DomainSpec) []string {
	domains := make([]string, len(specs))
	for i, s := range specs {
		domains[i] = s.Domain
	}
	return domains
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
		domains := specDomains(result)
		if len(domains) != len(test.expected) {
			t.Errorf("Expected %d permutations, got %d", len(test.expected), len(domains))
		}
		for _, perm := range test.expected {
			if !slices.Contains(domains, perm) {
				t.Errorf("Expected permutation %s not found in result", perm)
			}
		}
	}
}

func TestGenerateDomainPermutations_MetadataFields(t *testing.T) {
	app := config.NewTldxContext()
	s := composer.NewComposerService(app)
	app.Config.TLDs = []string{"com", "io"}
	app.Config.Prefixes = []string{"get"}
	app.Config.Suffixes = []string{"ly"}

	specs, warnings := s.GenerateDomainPermutations([]string{"stripe"})
	assert.Empty(t, warnings)
	assert.NotEmpty(t, specs)

	byDomain := make(map[string]resolver.DomainSpec)
	for _, spec := range specs {
		byDomain[spec.Domain] = spec
	}

	// bare keyword
	assert.Equal(t, "stripe", byDomain["stripe.com"].Keyword)
	assert.Equal(t, "", byDomain["stripe.com"].Prefix)
	assert.Equal(t, "", byDomain["stripe.com"].Suffix)
	assert.Equal(t, "com", byDomain["stripe.com"].TLD)

	// with prefix only
	assert.Equal(t, "stripe", byDomain["getstripe.com"].Keyword)
	assert.Equal(t, "get", byDomain["getstripe.com"].Prefix)
	assert.Equal(t, "", byDomain["getstripe.com"].Suffix)

	// with suffix only
	assert.Equal(t, "stripe", byDomain["stripely.com"].Keyword)
	assert.Equal(t, "", byDomain["stripely.com"].Prefix)
	assert.Equal(t, "ly", byDomain["stripely.com"].Suffix)

	// with prefix + suffix
	assert.Equal(t, "stripe", byDomain["getstripely.io"].Keyword)
	assert.Equal(t, "get", byDomain["getstripely.io"].Prefix)
	assert.Equal(t, "ly", byDomain["getstripely.io"].Suffix)
	assert.Equal(t, "io", byDomain["getstripely.io"].TLD)
}

