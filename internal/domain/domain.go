package domain

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
	"golang.org/x/net/publicsuffix"
)

type ConfigOptions struct {
	TLDs            []string
	Prefixes        []string
	Suffixes        []string
	MaxDomainLength int
	Verbose         bool
	OnlyAvailable   bool
	ShowStats       bool
}

type DomainResult struct {
	Domain    string `json:"domain"`
	Available bool   `json:"available"`
	Error     error  `json:"error,omitempty"`
}

type Stats struct {
	total        int
	available    int
	notAvailable int
	timedOut     int
	errored      int
}

const (
	maxRetries       = 3
	initialBackoff   = 500 * time.Millisecond
	backoffFactor    = 5.0
	jitterFraction   = 0.7 // +/-70% randomness
	contextTimeout   = 15 * time.Second
	concurrencyLimit = 20
)

var Config = ConfigOptions{}

var stats = Stats{}

func Exec(domainsOrKeywords []string) {
	keywords := validateKeywords(domainsOrKeywords)
	domains := generateDomainPermutations(keywords)
	stats.total = len(domains)
	resultChan := checkDomainsStreaming(domains, concurrencyLimit, contextTimeout)

	for result := range resultChan {
		if result.Error != nil {
			stats.errored += 1
			if Config.Verbose {
				fmt.Println(Errored(result.Domain, result.Error))
			}
			continue
		}
		if result.Available {
			stats.available += 1
			fmt.Println(Available(result.Domain))
		} else {
			stats.notAvailable += 1
			if Config.OnlyAvailable {
				continue
			}
			fmt.Println(NotAvailable(result.Domain))
		}
	}
	if Config.ShowStats {
		fmt.Println(RenderStatsSummary())
	}
}

func checkAvailability(ctx context.Context, domain string) (bool, error) {
	if !isValidDomainOrKeyword(domain) {
		return false, errors.New("Invalid domain")
	}

	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
			time.Sleep(time.Duration(float64(backoff) * (1 + (rand.Float64()*2-1)*jitterFraction)))
			raw, err := whois.Whois(domain)
			if err != nil || isTransientWhoisError(err, raw) {
				lastErr = err
				if attempt < maxRetries {
					jitter := time.Duration(float64(backoff) * (1 + (rand.Float64()*2-1)*jitterFraction))
					time.Sleep(jitter)
					backoff = time.Duration(float64(backoff) * backoffFactor)
					continue
				}
				stats.timedOut += 1
				return false, fmt.Errorf("whois error after %d retries: %w", attempt, err)
			}

			parsed, err := whoisparser.Parse(raw)
			if err != nil && strings.Contains(err.Error(), "domain is not found") {
				return true, nil
			}
			if parsed.Registrar != nil {
				return false, nil
			}
			return false, err
		}
	}

	return false, fmt.Errorf("unreachable: exhausted retries, last error: %v", lastErr)
}

func isTransientWhoisError(err error, raw string) bool {
	if err == nil && raw == "" {
		return true // empty response
	}
	if err != nil {
		msg := err.Error()
		return strings.Contains(msg, "connection reset") ||
			strings.Contains(msg, "timeout") ||
			strings.Contains(msg, "EOF") ||
			strings.Contains(msg, "refused") ||
			strings.Contains(msg, "too many requests")
	}
	return false
}

func checkDomainsStreaming(domains []string, concurrency int, timeout time.Duration) <-chan DomainResult {
	resultChan := make(chan DomainResult)
	inputChan := make(chan string)

	go func() {
		defer close(inputChan)
		for _, domain := range domains {
			inputChan <- domain
		}
	}()

	go func() {
		var wg sync.WaitGroup
		sem := make(chan struct{}, concurrency)

		for domain := range inputChan {
			domain := domain
			sem <- struct{}{}
			wg.Add(1)
			time.Sleep(50 * time.Millisecond) // Throttle requests to avoid rate limiting

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
		close(resultChan)
	}()

	return resultChan
}

func generateDomainPermutations(keywords []string) []string {
	var result []string
	var tlds []string

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
