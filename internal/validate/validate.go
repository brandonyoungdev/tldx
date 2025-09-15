package validate

import (
	"regexp"
	"strings"

	"github.com/brandonyoungdev/tldx/internal/strutil"
	"golang.org/x/net/publicsuffix"
)

type ValidatedKeywords struct {
	Keywords []string
	NewTlds  []string
}

// Returns a list of keywords or domains that are valid and have no duplicates.
// It also extracts TLDs from full domains and adds them to the config, if any.
func ValidateKeywords(domainsOrKeywords []string) *ValidatedKeywords {
	domainsOrKeywords = strutil.RemoveDuplicates(domainsOrKeywords)
	validatedKeywords := []string{}
	newTlds := []string{}
	for _, domainOrKeyword := range domainsOrKeywords {
		if !IsValidDomainOrKeyword(domainOrKeyword) {
			continue
		}

		// check if the domain entered has a TLD
		if strings.Contains(domainOrKeyword, ".") {

			tld, _ := publicsuffix.PublicSuffix(strings.ToLower(domainOrKeyword))

			domainOrKeyword = strings.TrimSuffix(domainOrKeyword, "."+tld)

			newTlds = append(newTlds, tld)
		}
		validatedKeywords = append(validatedKeywords, strings.ToLower(domainOrKeyword))
	}

	return &ValidatedKeywords{
		Keywords: strutil.RemoveDuplicates(validatedKeywords),
		NewTlds:  newTlds, // these are tlds found in keywords that we will want to consider
	}
}

var labelRegexp = regexp.MustCompile(`^(?i)[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

func IsValidDomainOrKeyword(domainOrKeyword string) bool {
	// Split domain into labels
	labels := strings.SplitSeq(domainOrKeyword, ".")
	for label := range labels {
		if !labelRegexp.MatchString(label) {
			return false
		}
	}

	return true
}
