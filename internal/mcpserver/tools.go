package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/brandonyoungdev/tldx/internal/composer"
	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/domain"
	"github.com/brandonyoungdev/tldx/internal/resolver"
	"github.com/brandonyoungdev/tldx/internal/validate"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Service) handleCheckDomains(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domains, err := req.RequireStringSlice("domains")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if len(domains) == 0 {
		return mcp.NewToolResultError("domains list must not be empty"), nil
	}
	if len(domains) > MaxDomainsPerCall {
		return mcp.NewToolResultError(fmt.Sprintf(
			"%d domains exceeds the %d-domain limit for one call. Split the list across several calls.",
			len(domains), MaxDomainsPerCall)), nil
	}

	var invalid []string
	specs := make([]resolver.DomainSpec, 0, len(domains))
	for _, d := range domains {
		if !validate.IsValidDomainOrKeyword(d) {
			invalid = append(invalid, d)
			continue
		}
		specs = append(specs, resolver.DomainSpec{Domain: d})
	}
	if len(specs) == 0 {
		return mcp.NewToolResultError(fmt.Sprintf("no valid domain names given: %v", invalid)), nil
	}

	app := s.context()
	applyForSaleArgs(app.Config, req)

	resp := s.collect(ctx, app, specs, 0)
	if len(invalid) > 0 {
		resp.Note = joinNotes(resp.Note, fmt.Sprintf("Skipped %d malformed name(s): %v.", len(invalid), invalid))
	}

	return toolResult(resp)
}

func (s *Service) handleGenerateAndCheck(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	keywords, err := req.RequireStringSlice("keywords")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if len(keywords) == 0 {
		return mcp.NewToolResultError("keywords list must not be empty"), nil
	}

	app := s.context()
	cfg := app.Config

	if v, ok := req.GetArguments()["tlds"]; ok && v != nil {
		cfg.TLDs = req.GetStringSlice("tlds", nil)
	}
	if v, ok := req.GetArguments()["prefixes"]; ok && v != nil {
		cfg.Prefixes = req.GetStringSlice("prefixes", nil)
	}
	if v, ok := req.GetArguments()["suffixes"]; ok && v != nil {
		cfg.Suffixes = req.GetStringSlice("suffixes", nil)
	}
	if v := req.GetString("tld_preset", ""); v != "" {
		cfg.TLDPreset = v
	}
	cfg.OnlyAvailable = req.GetBool("only_available", cfg.OnlyAvailable)
	if n := req.GetInt("max_domain_length", 0); n > 0 {
		cfg.MaxDomainLength = n
	}
	applyForSaleArgs(cfg, req)

	limit := req.GetInt("limit", cfg.Limit)
	dryRun := req.GetBool("dry_run", false)
	preset := cfg.TLDPreset

	specs, warnings := composer.NewComposerService(app).Compile(keywords)
	if len(specs) == 0 {
		msgs := make([]string, 0, len(warnings))
		for _, w := range warnings {
			msgs = append(msgs, w.Error())
		}
		return mcp.NewToolResultError(fmt.Sprintf(
			"these arguments produced no domains to check. %s", strings.Join(msgs, "; "))), nil
	}

	// Counted post-compile, where preset expansion, deduplication and
	// max_domain_length have already been applied.
	if len(specs) > MaxDomainsPerCall && !dryRun && limit <= 0 {
		return mcp.NewToolResultError(overBudgetMessage(specs, preset)), nil
	}

	if dryRun {
		return toolResult(dryRunResponse(specs, warnings))
	}

	resp := s.collect(ctx, app, specs, limit)
	resp.Note = joinNotes(resp.Note, warningNote(warnings))

	return toolResult(resp)
}

// collect runs the specs and folds the stream into one response, shared by both
// tools.
func (s *Service) collect(ctx context.Context, app *config.TldxContext, specs []resolver.DomainSpec, limit int) CheckResponse {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	deadline := time.NewTimer(softDeadline)
	defer deadline.Stop()

	resultChan := s.newResolver(app).CheckDomainsStreaming(ctx, specs)

	resp := CheckResponse{Results: []DomainCheck{}}
	availableFound := 0

collect:
	for {
		select {
		case r, ok := <-resultChan:
			if !ok {
				break collect
			}

			resp.Checked++
			switch {
			case r.Error != nil:
				resp.Errored++
			case r.Available:
				resp.Available++
			default:
				resp.Taken++
			}
			if r.ForSale != nil {
				resp.ForSale++
			}

			if domain.ShouldDisplay(app.Config, r) {
				resp.Results = append(resp.Results, fromResult(r))
			}

			if r.Available {
				availableFound++
				if limit > 0 && availableFound >= limit {
					cancel()
					break collect
				}
			}

		case <-deadline.C:
			resp.Truncated = true
			resp.Note = fmt.Sprintf(
				"Stopped after %s with %d of %d domains checked, to stay inside the client timeout. "+
					"Re-run with fewer TLDs or keywords, or set only_available=true with a limit.",
				softDeadline, resp.Checked, len(specs))
			cancel()
			break collect
		}
	}

	if !resp.Truncated && resp.Checked < len(specs) {
		resp.Truncated = true
		if limit > 0 && availableFound >= limit {
			resp.Note = fmt.Sprintf("Stopped early: reached limit of %d available domain(s) after checking %d of %d.",
				limit, resp.Checked, len(specs))
		} else {
			resp.Note = fmt.Sprintf("Stopped early after %d of %d domains.", resp.Checked, len(specs))
		}
	}

	return resp
}

func dryRunResponse(specs []resolver.DomainSpec, warnings []error) CheckResponse {
	domains := make([]string, 0, len(specs))
	for _, spec := range specs {
		domains = append(domains, spec.Domain)
	}

	resp := CheckResponse{
		Results: []DomainCheck{},
		DryRun:  true,
		Planned: len(specs),
		Domains: domains,
	}

	note := fmt.Sprintf("Nothing was checked. A real call with these arguments would resolve %d domain(s).", len(specs))
	if len(specs) > MaxDomainsPerCall {
		note += fmt.Sprintf(" That is over the %d-domain limit, so it would be rejected unless you set a limit.", MaxDomainsPerCall)
	}
	resp.Note = joinNotes(note, warningNote(warnings))

	return resp
}

// overBudgetMessage explains a rejection: the real count, how it factorises,
// and a concrete smaller call to make instead.
func overBudgetMessage(specs []resolver.DomainSpec, preset string) string {
	count := len(specs)
	keywords, tlds := countDistinct(specs)
	variants := count / max(keywords*tlds, 1) // prefix/suffix combinations per keyword

	var b strings.Builder
	fmt.Fprintf(&b, "%d domains exceeds the %d-domain limit for one call, so nothing was checked.\n",
		count, MaxDomainsPerCall)

	tldSource := fmt.Sprintf("%d TLDs", tlds)
	if preset != "" {
		tldSource = fmt.Sprintf("%d TLDs (tld_preset %q)", tlds, preset)
	}
	fmt.Fprintf(&b, "Breakdown: %d keyword(s) x %d variant(s) each x %s.\n", keywords, variants, tldSource)

	perTLD := max(count/max(tlds, 1), 1)
	perKeyword := max(count/max(keywords, 1), 1)

	// Only advise shrinking a dimension when doing so gets under the cap.
	tldFixes := tlds > 1 && perTLD*2 <= MaxDomainsPerCall
	keywordFixes := keywords > 1 && perKeyword <= MaxDomainsPerCall

	switch {
	case tldFixes && tlds >= keywords && tlds >= variants:
		fit := min(MaxDomainsPerCall/perTLD, tlds-1)
		fmt.Fprintf(&b, "Easiest fix: shorten the TLD list, at about %d domain(s) per TLD. "+
			"Up to %d TLDs fit; tlds=[\"com\",\"io\",\"ai\"] is about %d domains.\n",
			perTLD, fit, perTLD*min(3, fit))

	case keywordFixes && keywords >= variants:
		fit := max(MaxDomainsPerCall/perKeyword, 1)
		calls := (keywords + fit - 1) / fit
		fmt.Fprintf(&b, "Easiest fix: split the keywords, at about %d domain(s) per keyword. "+
			"Send at most %d per call; %d calls covers all %d.\n",
			perKeyword, fit, calls, keywords)

	case variants > 1:
		fmt.Fprintf(&b, "Easiest fix: prefixes and suffixes multiply every keyword by %d. "+
			"Dropping the suffixes would cut the count to roughly %d.\n",
			variants, count/variants*(1+(variants-1)/2))

	case tldFixes:
		fit := min(MaxDomainsPerCall/perTLD, tlds-1)
		fmt.Fprintf(&b, "Easiest fix: shorten the TLD list to at most %d TLD(s).\n", fit)

	case keywordFixes:
		fit := max(MaxDomainsPerCall/perKeyword, 1)
		fmt.Fprintf(&b, "Easiest fix: send at most %d keyword(s) per call.\n", fit)

	default:
		b.WriteString("Every dimension is already small; lower max_domain_length or split the call.\n")
	}

	b.WriteString("Or keep every argument and set only_available=true with limit=N " +
		"(e.g. limit=20): the sweep stops at N available domains, so the full space is affordable. " +
		"Use dry_run=true to preview a count for free.")

	return b.String()
}

// countDistinct reads the dimensions back off the compiled specs, the only
// source reflecting preset expansion, deduplication and length filtering.
func countDistinct(specs []resolver.DomainSpec) (keywords, tlds int) {
	seenKeyword := map[string]struct{}{}
	seenTLD := map[string]struct{}{}
	for _, spec := range specs {
		seenKeyword[spec.Keyword] = struct{}{}
		seenTLD[spec.TLD] = struct{}{}
	}
	return len(seenKeyword), len(seenTLD)
}

func applyForSaleArgs(cfg *config.TldxConfigOptions, req mcp.CallToolRequest) {
	cfg.CheckForSale = req.GetBool("check_for_sale", cfg.CheckForSale)
	cfg.OnlyForSale = req.GetBool("only_for_sale", cfg.OnlyForSale)
	// Mirrors cmd/root.go.
	if cfg.OnlyForSale {
		cfg.CheckForSale = true
	}
}

func warningNote(warnings []error) string {
	if len(warnings) == 0 {
		return ""
	}
	msgs := make([]string, 0, len(warnings))
	for _, w := range warnings {
		msgs = append(msgs, w.Error())
	}
	return strings.Join(msgs, "; ")
}

func joinNotes(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, " ")
}

func toolResult(resp CheckResponse) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultStructuredOnly(resp), nil
}
