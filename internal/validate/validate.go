package validate

import (
	"regexp"
	"strings"

	"github.com/brandonyoungdev/tldx/internal/strutil"
)

// Returns a list of keywords or domains that are valid and have no duplicates.
func ValidateKeywords(domainsOrKeywords []string) []string {
	validatedKeywords := []string{}
	for _, domainOrKeyword := range domainsOrKeywords {
		if !IsValidDomainOrKeyword(domainOrKeyword) {
			continue
		}
		validatedKeywords = append(validatedKeywords, strings.ToLower(domainOrKeyword))
	}
	return strutil.RemoveDuplicates(validatedKeywords)
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
