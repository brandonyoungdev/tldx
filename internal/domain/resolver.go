package domain

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/openrdap/rdap"
)

type ResolverService struct {
	rdapClient *rdap.Client
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
		}, nil
	}

	if domainResponse == nil {
		fmt.Println("RDAP response is nil for domain:", domain)
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

func (s *ResolverService) checkDNS(domain string) (CheckResult, error) {
	ips, err := net.LookupHost(domain)
	if err != nil {
		// DNS resolution failure: might be unregistered or inactive
		return CheckResult{Registered: false}, err
	}
	return CheckResult{
		Registered: true,
		Details:    fmt.Sprintf("Resolved to: %v", ips),
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

	resp, err := s.rdapClient.Do(req)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch RDAP data for domain %q: %w", domain, err)
	}

	if _, ok := resp.Object.(*rdap.Domain); !ok {
		return nil, fmt.Errorf("unexpected RDAP object type for domain %q", domain)
	}

	return resp.Object.(*rdap.Domain), nil
}
