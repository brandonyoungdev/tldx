package composer

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/presets"
	"github.com/brandonyoungdev/tldx/internal/regex"
	"github.com/brandonyoungdev/tldx/internal/resolver"
	"github.com/brandonyoungdev/tldx/internal/strutil"
	"github.com/brandonyoungdev/tldx/internal/validate"
	"golang.org/x/net/publicsuffix"
)

type ComposerService struct {
	app *config.TldxContext
}

func NewComposerService(app *config.TldxContext) *ComposerService {
	return &ComposerService{
		app,
	}
}

func (s *ComposerService) Compile(domainsOrKeywords []string) ([]resolver.DomainSpec, []error) {
	domainsOrKeywords = strutil.AllToLowerCase(domainsOrKeywords)

	// Expand regex patterns if regex mode is enabled
	if s.app.Config.Regex {
		expanded, err := s.expandRegexPatterns(domainsOrKeywords)
		if err != nil {
			return nil, []error{err}
		}
		domainsOrKeywords = expanded
	}

	validatedKeywords := validate.ValidateKeywords(domainsOrKeywords)

	// Add any new TLDs found in keywords to the config
	s.app.Config.TLDs = append(s.app.Config.TLDs, validatedKeywords.NewTlds...)

	specs, warnings := s.GenerateDomainPermutations(validatedKeywords.Keywords)

	if s.app.Config.MaxDomainLength > 0 {
		filtered := specs[:0]
		for _, spec := range specs {
			if len(spec.Domain) <= s.app.Config.MaxDomainLength {
				filtered = append(filtered, spec)
			}
		}
		specs = filtered
	}

	return specs, warnings
}

func (s *ComposerService) GenerateDomainPermutations(keywords []string) ([]resolver.DomainSpec, []error) {
	var result []resolver.DomainSpec
	var tlds []string
	var warnings []error

	for _, tld_candidate := range s.app.Config.TLDs {
		tld, ok := publicsuffix.PublicSuffix(strings.ToLower(tld_candidate))
		if !ok {
			warnings = append(warnings, fmt.Errorf("%v: invalid TLD", tld_candidate))
			continue
		}
		tlds = append(tlds, tld)
	}
	tlds = strutil.RemoveDuplicates(tlds)
	s.app.Config.TLDs = tlds

	if s.app.Config.TLDPreset != "" {
		// Strip out any . from the preset name
		tldPreset := strings.TrimPrefix(s.app.Config.TLDPreset, ".")

		var additionalTlds []string
		if tldPreset == "all" {
			additionalTlds = presets.GetAllTLDs()
		} else if tlds, ok := presets.TLDs.Get(tldPreset); ok {
			additionalTlds = tlds
		} else {
			warnings = append(warnings, fmt.Errorf("Error: TLD preset not found"))
		}
		tlds = append(tlds, additionalTlds...)
	}

	if len(tlds) == 0 {
		tlds = []string{"com"} // Default TLDs if none provided
	}

	prefixes := s.app.Config.Prefixes
	suffixes := s.app.Config.Suffixes

	type combo struct{ prefix, suffix string }

	seen := make(map[string]bool)

	for _, keyword := range keywords {
		combos := []combo{{"", ""}}

		for _, prefix := range prefixes {
			combos = append(combos, combo{prefix, ""})
			for _, suffix := range suffixes {
				combos = append(combos, combo{prefix, suffix})
			}
		}
		for _, suffix := range suffixes {
			combos = append(combos, combo{"", suffix})
		}

		for _, c := range combos {
			base := c.prefix + keyword + c.suffix
			for _, tld := range tlds {
				domain := fmt.Sprintf("%s.%s", base, tld)
				if seen[domain] {
					continue
				}
				seen[domain] = true
				result = append(result, resolver.DomainSpec{
					Domain:  domain,
					Keyword: keyword,
					Prefix:  c.prefix,
					Suffix:  c.suffix,
					TLD:     tld,
				})
			}
		}
	}

	return result, warnings
}

func (s *ComposerService) expandRegexPatterns(keywords []string) ([]string, error) {
	const maxCombinations = 500000
	expanded := make([]string, 0, len(keywords))

	for _, keyword := range keywords {
		if !isRegexPattern(keyword) {
			expanded = append(expanded, keyword)
			continue
		}

		if err := s.validateAndExpandPattern(keyword, maxCombinations, &expanded); err != nil {
			return nil, err
		}
	}

	return expanded, nil
}

func (s *ComposerService) validateAndExpandPattern(pattern string, maxCombinations int, expanded *[]string) error {
	safe, count, err := regex.IsPatternSafe(pattern, maxCombinations)
	if err != nil {
		return fmt.Errorf("invalid regex pattern '%s': %w", pattern, err)
	}

	if !safe {
		s.logUnsafePattern(pattern, count)
		return nil // Skip unsafe patterns but don't error
	}

	results, err := regex.ExpandPattern(pattern)
	if err != nil {
		return fmt.Errorf("failed to expand regex pattern '%s': %w", pattern, err)
	}

	*expanded = append(*expanded, results...)

	return nil
}

func (s *ComposerService) logUnsafePattern(pattern string, count int) {
	if !s.app.Config.Verbose {
		return
	}

	slog.Warn(fmt.Sprintf("Pattern '%s' would generate more than 500,000 combinations (%d). Skipping for safety.", pattern, count))

}

func isRegexPattern(s string) bool {
	return strings.Contains(s, "[") || strings.Contains(s, "{") || strings.Contains(s, "\\")
}
