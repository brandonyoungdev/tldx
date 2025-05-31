package domain

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
)

type Options struct {
	TLDs            []string
	Prefixes        []string
	Suffixes        []string
	MaxDomainLength int
}

type DomainResult struct {
	Domain    string `json:"domain"`
	Available bool   `json:"available"`
	Error     error  `json:"error,omitempty"`
}

var Config = Options{}

func Exec(domainsOrKeywords []string) {
	keywords := validateKeywords(domainsOrKeywords)
	domains := generateDomainPermutations(keywords)
	resultChan := checkDomainsStreaming(domains, 20, 15*time.Second)

	for result := range resultChan {
		if result.Error != nil {
			fmt.Println(Errored(result.Domain, result.Error))
			continue
		}
		if result.Available {
			fmt.Println(Available(result.Domain))
		} else {
			fmt.Println(NotAvailable(result.Domain))
		}
	}
}

// CheckAvailability checks if a domain is available with timeout support.
func checkAvailability(ctx context.Context, domain string) (bool, error) {
	if !isValidDomainOrKeyword(domain) {
		return false, errors.New("Invalid domain")
	}

	type whoisResult struct {
		raw string
		err error
	}

	resultCh := make(chan whoisResult, 1)
	go func() {
		raw, err := whois.Whois(domain)
		resultCh <- whoisResult{raw: raw, err: err}
	}()

	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case res := <-resultCh:
		if res.err != nil {
			return false, res.err
		}

		parsed, err := whoisparser.Parse(res.raw)
		if err != nil && strings.Contains(err.Error(), "domain is not found") {
			return true, nil
		}
		if parsed.Registrar != nil {
			return false, nil
		}
		return false, err
	}
}

// WorkerPool to check domains concurrently with timeouts and streaming output.
func checkDomainsStreaming(domains []string, concurrency int, timeout time.Duration) <-chan DomainResult {
	resultChan := make(chan DomainResult)

	go func() {
		defer close(resultChan)
		sem := make(chan struct{}, concurrency)
		var wg sync.WaitGroup

		for _, domain := range domains {
			domain := domain // capture loop variable correctly
			sem <- struct{}{}
			wg.Add(1)

			go func() {
				defer func() {
					<-sem
					wg.Done()
				}()

				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				defer cancel()

				available, err := checkAvailability(ctx, domain)
				resultChan <- DomainResult{
					Domain:    domain,
					Available: available,
					Error:     err,
				}
			}()
		}

		wg.Wait()
	}()

	return resultChan
}

func generateDomainPermutations(keywords []string) []string {
	var result []string
	tlds := Config.TLDs

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
			// This is a full domain, let's remove the tld and add it to our config
			tld := strings.Split(domainOrKeyword, ".")
			domainOrKeyword = strings.Join(tld[:len(tld)-1], ".")
			Config.TLDs = append(Config.TLDs, tld[len(tld)-1])
		}
		validatedKeywords = append(validatedKeywords, domainOrKeyword)
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
