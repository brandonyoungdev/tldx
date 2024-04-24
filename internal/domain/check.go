package domain

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
)

func CheckAvailability(domain string) (bool, error) {
	// validate that the domain is a proper domain.
	if !(isValidDomain(domain) && strings.Contains(domain, ".")) {
		return false, errors.New("Invalid domain")
	}

	whoisRaw, err := whois.Whois(domain)
	if err != nil {
		slog.Error(fmt.Sprintf("Error checking whois for domain: %s : %s", domain, err))
		return false, err
	}

	// parse Whois result
	result, err := whoisparser.Parse(whoisRaw)
	if err != nil {
		if strings.Contains(err.Error(), "domain is not found") {
			// slog.Info(fmt.Sprintf("Domain: %s is available", domain))
			return true, nil
		}
	}

	if result.Registrar != nil {
		// slog.Info(fmt.Sprintf("Registrar: %s", result.Registrar))
		return false, nil
	}

	slog.Info("Registrar: Not found")

	return true, nil
}

func RetryCheckAvailability(domain string, retries int, delay time.Duration) (bool, error) {
	var previousError error
	for i := 0; i < retries; i++ {
		available, err := CheckAvailability(domain)
		if err != nil {
			slog.Error(fmt.Sprintf("Error checking domain: %s. Retrying... %s", domain, err))
			previousError = err
      // wait a small amount of time to prevent rate limiting based on which retry we are on
      time.Sleep(time.Duration(delay.Seconds() * float64(i)))
			continue
		}
		return available, nil
	}
	if previousError != nil {
		previousError = errors.New(fmt.Sprintf("Failed to check domain %s after %d retries", domain, retries))
	}
	return false, previousError
}

func CheckDomains(domains []string) (map[string]bool, map[string]error) {
  validatedDomains := []string{}
  for _, domain := range domains {

    // validate that the domain is a proper domain.
    if isValidDomain(domain) && strings.Contains(domain, "."){
      validatedDomains = append(validatedDomains, domain)
    }
  }
  domains = validatedDomains

	// remove duplicates from domains
	domains = removeDuplicates(domains)

	var wg sync.WaitGroup
	results := make(map[string]bool)
	errors := make(map[string]error)

	resultsChan := make(chan map[string]bool, len(domains))
	errorChan := make(chan map[string]error, len(domains))

	worker := func(domain string) {
		defer wg.Done()
		available, err := RetryCheckAvailability(domain, 3, 3*time.Second)
		if err != nil {
			errorChan <- map[string]error{domain: err}
			return
		}
		resultsChan <- map[string]bool{domain: available}
	}

	wg.Add(len(domains))
	for _, domain := range domains {
		// wait a small amount of time to prevent rate limiting
		time.Sleep(50 * time.Millisecond)
		go worker(domain)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
		close(errorChan)
	}()

	for res := range resultsChan {
		for k, v := range res {
			results[k] = v
		}
	}

	for err := range errorChan {
		for k, v := range err {
			errors[k] = v
		}
	}
	return results, errors
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

func CheckAndPrint(domains []string) {

	results, errs := CheckDomains(domains)
	for erroredDomain, err := range errs {
		Errored(erroredDomain, err)
	}

	for result, available := range results {
		if available {
			fmt.Println(Available(result))
		} else {
			fmt.Println(NotAvailable(result))
		}
	}
}

func isValidDomain(domain string) bool {
	// Check overall length of domain
	if len(domain) > 253 {
		return false
	}

	// Regular expression to validate each label
	labelRegexp := regexp.MustCompile(`^(?i)[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

	// Split domain into labels
	labels := strings.Split(domain, ".")
	for _, label := range labels {
		if !labelRegexp.MatchString(label) || len(label) > 63 {
			return false
		}
	}

	return true
}
