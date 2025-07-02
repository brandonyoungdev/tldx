package resolver

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/validate"
	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
	"github.com/openrdap/rdap"
	"github.com/openrdap/rdap/bootstrap"
)

type ResolverService struct {
	httpClient *http.Client
	app        *config.TldxContext
}

type DomainResult struct {
	Domain    string `json:"domain"`
	Available bool   `json:"available"`
	Details   string `json:"details,omitempty"`
	Error     error  `json:"error,omitempty"`
}

type EncodableDomainResult struct {
	Domain    string `json:"domain"`
	Available bool   `json:"available"`
	Details   string `json:"details,omitempty"`
	Error     string `json:"error,omitempty"`
}

type CheckResult struct {
	Registered bool
	Details    string
}

func (result DomainResult) AsEncodable() EncodableDomainResult {
	errMsg := ""
	if result.Error != nil {
		errMsg = result.Error.Error()
	}
	return EncodableDomainResult{
		Domain:    result.Domain,
		Available: result.Available,
		Details:   result.Details,
		Error:     errMsg,
	}
}

type Resolver interface {
	Check(domain string) (*CheckResult, error)
}

func NewResolverService(app *config.TldxContext) *ResolverService {
	return &ResolverService{
		app:        app,
		httpClient: &http.Client{},
	}
}

func (s *ResolverService) withRetry(ctx context.Context, fn func() (CheckResult, error)) (CheckResult, error) {
	var lastErr error
	backoff := s.app.Config.InitialBackoff
	maxBackoff := s.app.Config.MaxBackoff

	for attempt := 0; attempt <= s.app.Config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return CheckResult{}, ctx.Err()
		default:
			result, err := fn()
			if err == nil {
				return result, nil
			}

			if !isRetryable(err) || attempt == s.app.Config.MaxRetries {
				return CheckResult{}, err
			}

			lastErr = err

			sleep := time.Duration(rand.Float64() * float64(backoff))
			select {
			case <-time.After(sleep):
				// Sleep completed
			case <-ctx.Done():
				return CheckResult{}, ctx.Err()
			}

			backoff = min(time.Duration(float64(backoff)*s.app.Config.BackoffFactor), maxBackoff)
		}
	}

	return CheckResult{}, lastErr
}

var retryablePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)timeout`),
	regexp.MustCompile(`(?i)connection.*refused`),
	regexp.MustCompile(`(?i)temporary`),
	regexp.MustCompile(`(?i)i/o timeout`),
	regexp.MustCompile(`(?i)responded successfully`),
}

func isRetryable(err error) bool {
	errStr := err.Error()
	for _, r := range retryablePatterns {
		if r.MatchString(errStr) {
			return true
		}
	}
	return false
}

func (s *ResolverService) CheckDomain(ctx context.Context, domain string) (CheckResult, error) {
	if !validate.IsValidDomainOrKeyword(domain) {
		return CheckResult{}, errors.New("invalid domain")
	}

	rdapResult, err := s.withRetry(ctx, func() (CheckResult, error) {
		return s.checkRDAP(ctx, domain)
	})
	if err == nil {
		return rdapResult, nil
	}

	if strings.Contains(err.Error(), "No RDAP servers found for") {
		// dns fallback
		dnsResolved, _ := s.checkIfDNSResolves(ctx, domain)

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
				Details:    fmt.Sprintf("RDAP is not found or doesn't exist"),
			}, nil
		}

		return CheckResult{
			Registered: true,
			Details:    fmt.Sprintf("RDAP query error"),
		}, err
	}

	if domainResponse == nil {
		return CheckResult{
			Registered: false,
			Details:    fmt.Sprintf("No RDAP response available"),
		}, nil
	}

	return CheckResult{
		Registered: true,
		Details:    fmt.Sprintf("Rdap registered: %s", domainResponse.Status),
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

	registrar := "<unknown>"
	created := "<unknown>"

	if parsed.Registrar != nil && parsed.Registrar.Name != "" {
		registrar = parsed.Registrar.Name
	}
	if parsed.Domain != nil && parsed.Domain.CreatedDate != "" {
		created = parsed.Domain.CreatedDate
	}

	return CheckResult{
		Registered: true,
		Details:    fmt.Sprintf("WHOIS Registered: %s (%s)", registrar, created),
	}, nil
}

func (s *ResolverService) CheckDomainsStreaming(domains []string) <-chan DomainResult {
	resultChan := make(chan DomainResult)

	go func() {
		var wg sync.WaitGroup
		limit := s.app.Config.ConcurrencyLimit
		if limit <= 0 {
			limit = runtime.NumCPU()
		}
		sem := make(chan struct{}, limit)

		for _, domain := range domains {
			domain := domain // capture loop variable
			sem <- struct{}{}
			wg.Add(1)

			go func() {
				defer func() {
					<-sem
					wg.Done()
				}()

				ctx, cancel := context.WithTimeout(context.Background(), s.app.Config.ContextTimeout)
				defer cancel()

				checkResult, err := s.CheckDomain(ctx, domain)

				resultChan <- DomainResult{
					Domain:    domain,
					Available: !checkResult.Registered,
					Details:   checkResult.Details,
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
		Timeout: s.app.Config.ContextTimeout,
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
		return nil, fmt.Errorf("failed to fetch RDAP data: %w", err)
	}

	if _, ok := resp.Object.(*rdap.Domain); !ok {
		return nil, fmt.Errorf("unexpected RDAP object type")
	}

	return resp.Object.(*rdap.Domain), nil
}
