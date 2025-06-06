package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/brandonyoungdev/tldx/internal/presets"
	"golang.org/x/net/publicsuffix"
)

func Exec(domainsOrKeywords []string) {
	keywords := validateKeywords(domainsOrKeywords)
	domains := generateDomainPermutations(keywords)
	stats.total = len(domains)
	resolverService := NewResolverService()
	resultChan := resolverService.checkDomainsStreaming(domains, concurrencyLimit, contextTimeout)

	output := GetOutputWriter(Config.OutputFormat)

	for result := range resultChan {
		if result.Error != nil {
			stats.errored++
		} else if result.Available {
			stats.available++
		} else {
			stats.notAvailable++
		}
		output.Write(result)
	}

	output.Flush()

	if Config.ShowStats && Config.OutputFormat == "text" {
		fmt.Println(RenderStatsSummary())
	}
}

func generateDomainPermutations(keywords []string) []string {
	var result []string
	var tlds []string

	if Config.AllTLDs {
		tlds = presets.GetAllTLDs()
	}

	for _, tld_candidate := range Config.TLDs {
		tld, ok := publicsuffix.PublicSuffix(strings.ToLower(tld_candidate))
		if !ok {
			if !Config.OnlyAvailable {
				fmt.Println(Errored(tld_candidate, errors.New("invalid TLD")))
			}
			continue
		}
		tlds = append(tlds, tld)
	}
	tlds = removeDuplicates(tlds)
	Config.TLDs = tlds

	if Config.TLDPreset != "" && !Config.AllTLDs {
		// Strip out any . from the preset name
		tldPreset := strings.TrimPrefix(Config.TLDPreset, ".")

		store := presets.NewTypedStore("tld", presets.DefaultTLDPresets)
		additionalTlds, ok := store.Get(tldPreset)
		if !ok {
			fmt.Println("Error: TLD preset not found:", tldPreset)
		}
		tlds = append(tlds, additionalTlds...)

	}

	if len(tlds) == 0 {
		tlds = []string{"com"} // Default TLDs if none provided
	}

	prefixes := Config.Prefixes
	suffixes := Config.Suffixes

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

	return removeDuplicates(result)
}

// Returns a list of keywords or domains that are valid and have no duplicates.
// It also extracts TLDs from full domains and adds them to the config, if any.
func validateKeywords(domainsOrKeywords []string) []string {
	domainsOrKeywords = removeDuplicates(domainsOrKeywords)
	validatedKeywords := []string{}
	for _, domainOrKeyword := range domainsOrKeywords {
		if !isValidDomainOrKeyword(domainOrKeyword) {
			continue
		}

		// check if the domain entered has a TLD
		if strings.Contains(domainOrKeyword, ".") {

			tld, _ := publicsuffix.PublicSuffix(strings.ToLower(domainOrKeyword))

			domainOrKeyword = strings.TrimSuffix(domainOrKeyword, "."+tld)

			Config.TLDs = append(Config.TLDs, tld)
		}
		validatedKeywords = append(validatedKeywords, strings.ToLower(domainOrKeyword))
	}

	return removeDuplicates(validatedKeywords)
}

func isValidDomainOrKeyword(domainOrKeyword string) bool {
	// Check overall length of domain
	if len(domainOrKeyword) > Config.MaxDomainLength {
		return false
	}

	// Regular expression to validate each label
	labelRegexp := regexp.MustCompile(`^(?i)[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

	// Split domain into labels
	labels := strings.Split(domainOrKeyword, ".")
	for _, label := range labels {
		if !labelRegexp.MatchString(label) || len(label) > Config.MaxDomainLength {
			return false
		}
	}

	return true
}

func removeDuplicates(strs []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range strs {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func PrintDomain(domain string, available bool, err error) {
	if err != nil {
		fmt.Println(Errored(domain, err))
		return
	}
	if available {
		fmt.Println(Available(domain))
	} else {
		fmt.Println(NotAvailable(domain))
	}
}
