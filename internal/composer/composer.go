package composer

import (
	"errors"
	"fmt"
	"strings"

	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/presets"
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

func (s *ComposerService) Compile(domainsOrKeywords []string) ([]string, []error) {
	domainsOrKeywords = strutil.AllToLowerCase(domainsOrKeywords)
	validatedKeywords := validate.ValidateKeywords(domainsOrKeywords)

	// Add any new TLDs found in keywords to the config
	s.app.Config.TLDs = append(s.app.Config.TLDs, validatedKeywords.NewTlds...)

	domains, warnings := s.generateDomainPermutations(validatedKeywords.Keywords)

	if s.app.Config.MaxDomainLength > 0 {
		domains = strutil.FilterByMaxLength(domains, s.app.Config.MaxDomainLength)
	}

	return domains, warnings
}

func (s *ComposerService) generateDomainPermutations(keywords []string) ([]string, []error) {
	var result []string
	var tlds []string
	var warnings []error

	for _, tld_candidate := range s.app.Config.TLDs {
		tld, ok := publicsuffix.PublicSuffix(strings.ToLower(tld_candidate))
		if !ok {
			warnings = append(warnings, errors.New(fmt.Sprintf("%v: invalid TLD", tld_candidate)))
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
			warnings = append(warnings, errors.New(fmt.Sprintf("Error: TLD preset not found")))
		}
		tlds = append(tlds, additionalTlds...)
	}

	if len(tlds) == 0 {
		tlds = []string{"com"} // Default TLDs if none provided
	}

	prefixes := s.app.Config.Prefixes
	suffixes := s.app.Config.Suffixes

	for _, keyword := range keywords {
		bases := []string{keyword}

		// Generate permutations with prefixes and suffixes
		for _, prefix := range prefixes {
			bases = append(bases, prefix+keyword)
			for _, suffix := range suffixes {
				bases = append(bases, prefix+keyword+suffix)
			}
		}
		for _, suffix := range suffixes {
			bases = append(bases, keyword+suffix)
		}

		// Append TLDs to each base
		for _, base := range bases {
			for _, tld := range tlds {
				result = append(result, fmt.Sprintf("%s.%s", base, tld))
			}
		}
	}

	return strutil.RemoveDuplicates(result), warnings
}
