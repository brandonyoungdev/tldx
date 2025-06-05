package domain

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
	"github.com/openrdap/rdap"
	"github.com/openrdap/rdap/bootstrap"
)

type ResolverService struct {
	httpClient *http.Client
	config     *ConfigOptions
}

type DomainResult struct {
	Domain    string `json:"domain"`
	Available bool   `json:"available"`
	Error     error  `json:"error,omitempty"`
}

type CheckResult struct {
	Registered bool
	Details    string
}

type Resolver interface {
	Check(domain string) (*CheckResult, error)
}

func NewResolverService() *ResolverService {
	return &ResolverService{
		config:     &Config,
		httpClient: &http.Client{},
	}
}

func (s *ResolverService) withRetry(ctx context.Context, fn func() (CheckResult, error)) (CheckResult, error) {
	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return CheckResult{}, ctx.Err()
		default:
			result, err := fn()
			if err == nil {
				return result, nil
			}
			lastErr = err
			if attempt < maxRetries {
				jitter := time.Duration(float64(backoff) * (1 + (rand.Float64()*2-1)*jitterFraction))
				time.Sleep(jitter)
				backoff = time.Duration(float64(backoff) * backoffFactor)
			}
		}
	}

	return CheckResult{}, lastErr
}

func (s *ResolverService) CheckDomain(ctx context.Context, domain string) (CheckResult, error) {
	if !isValidDomainOrKeyword(domain) {
		return CheckResult{}, errors.New("invalid domain")
	}

	rdapResult, err := s.checkRDAP(ctx, domain)
	if err == nil {
		return rdapResult, nil
	}

	if strings.Contains(err.Error(), "No RDAP servers found for") {
		// dns fallback
		dnsResolved, err := s.checkIfDNSResolves(ctx, domain)
		if s.config.Verbose {
			fmt.Println("Checking DNS resolution for domain:", domain, "Resolved:", dnsResolved, "Error:", err)
		}

		if dnsResolved {
			return CheckResult{
				Registered: true,
				Details:    fmt.Sprintf("Domain %s has a DNS record, but RDAP is not available", domain),
			}, nil
		}

		whoisResult, err := s.checkWhois(ctx, domain)
		if !whoisResult.Registered && err == nil {
			return whoisResult, err
		}

	}

	if ctx.Err() != nil {
		return CheckResult{}, ctx.Err()
	}

	return CheckResult{
		Registered: false,
		Details:    "This domain has unknown status",
	}, fmt.Errorf("checkRDAP failed: %w", err)
}

func (s *ResolverService) checkRDAP(ctx context.Context, domain string) (CheckResult, error) {
	select {
	case <-ctx.Done():
		return CheckResult{
			Registered: false,
			Details:    fmt.Sprintf("Context cancelled before RDAP for %s", domain),
		}, ctx.Err()
	default:
		// continue
	}

	domainResponse, err := s.QueryDomainContext(ctx, domain)

	// name might be <nil> if no rdap found
	if err != nil {
		// check if the RDAP is not found (404)
		if strings.Contains(err.Error(), "object does not exist.") || strings.Contains(err.Error(), "404") {
			return CheckResult{
				Registered: false,
				Details:    fmt.Sprintf("RDAP is not found or doesn't exist for domain %s:", domain),
			}, nil
		}

		return CheckResult{
			Registered: true,
			Details:    fmt.Sprintf("RDAP query error for domain %s:", domain),
		}, err
	}

	if domainResponse == nil {
		if s.config.Verbose {
			fmt.Println("RDAP response is nil for domain:", domain)
		}
		return CheckResult{
			Registered: false,
			Details:    fmt.Sprintf("No RDAP available for %s", domain),
		}, nil
	}

	return CheckResult{
		Registered: true,
		Details:    fmt.Sprintf("Status: %s", domainResponse.Status),
	}, nil
}

func (s *ResolverService) checkIfDNSResolves(ctx context.Context, domain string) (bool, error) {
	resolver := net.Resolver{}
	ips, err := resolver.LookupHost(ctx, domain)
	if err != nil {
		return false, err
	}

	return len(ips) > 0, nil
}

func (s *ResolverService) checkWhois(ctx context.Context, domain string) (CheckResult, error) {
	type result struct {
		raw string
		err error
	}

	resultCh := make(chan result, 1)

	go func() {
		raw, err := whois.Whois(domain)
		resultCh <- result{raw: raw, err: err}
	}()

	var whoisRaw string
	select {
	case <-ctx.Done():
		return CheckResult{Registered: false}, ctx.Err()
	case res := <-resultCh:
		if res.err != nil {
			// Fallback: detect "not found" in raw whois text if err is nil but body says unregistered
			if strings.Contains(strings.ToLower(res.err.Error()), "no whois server") {
				return CheckResult{
					Registered: false,
					Details:    "WHOIS server not found for domain",
				}, nil
			}
			return CheckResult{Registered: false}, fmt.Errorf("WHOIS lookup error: %w", res.err)
		}
		whoisRaw = res.raw
	}

	parsed, err := whoisparser.Parse(whoisRaw)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "domain is not found") ||
			strings.Contains(strings.ToLower(whoisRaw), "no match for") ||
			strings.Contains(strings.ToLower(whoisRaw), "not found") {
			// Domain is likely unregistered
			return CheckResult{
				Registered: false,
				Details:    "Domain not registered (WHOIS says not found)",
			}, nil
		}

		return CheckResult{
			Registered: false,
			Details:    fmt.Sprintf("Failed to parse WHOIS for %s: %v", domain, err),
		}, nil
	}

	return CheckResult{
		Registered: true,
		Details: fmt.Sprintf("WHOIS Registered: %s (%s)",
			parsed.Registrar.Name, parsed.Domain.CreatedDate),
	}, nil
}

func (s ResolverService) checkDomainsStreaming(domains []string, concurrency int, timeout time.Duration) <-chan DomainResult {
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
			go func() {
				defer func() {
					<-sem
					wg.Done()
				}()

				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				defer cancel()

				available, err := s.CheckDomain(ctx, domain)
				resultChan <- DomainResult{
					Domain:    domain,
					Available: !available.Registered,
					Error:     err,
				}
			}()
		}

		wg.Wait()
		close(resultChan)
	}()

	return resultChan
}

func (s ResolverService) QueryDomainContext(ctx context.Context, domain string) (*rdap.Domain, error) {
	req := &rdap.Request{
		Type:    rdap.DomainRequest,
		Query:   domain,
		Timeout: contextTimeout,
	}

	req = req.WithContext(ctx)

	client := &rdap.Client{
		Bootstrap: &bootstrap.Client{
			HTTP: s.httpClient,
		},
		HTTP: s.httpClient,
	}

	resp, err := client.Do(req)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch RDAP data for domain %q: %w", domain, err)
	}

	if _, ok := resp.Object.(*rdap.Domain); !ok {
		return nil, fmt.Errorf("unexpected RDAP object type for domain %q", domain)
	}

	return resp.Object.(*rdap.Domain), nil
}
