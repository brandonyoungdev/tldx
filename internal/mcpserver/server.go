// Package mcpserver exposes tldx over the Model Context Protocol.
package mcpserver

import (
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/presets"
	"github.com/brandonyoungdev/tldx/internal/resolver"
	"github.com/brandonyoungdev/tldx/internal/userconfig"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	// MaxDomainsPerCall bounds how many domains one tool call may resolve.
	MaxDomainsPerCall = 1000

	// softDeadline returns partial results ahead of a typical 60s client timeout.
	softDeadline = 45 * time.Second

	// Mirrors the --max-domain-length default in cmd/root.go.
	defaultMaxDomainLength = 64
)

type ResolverFactory func(app *config.TldxContext, opts ...resolver.ResolverOption) *resolver.ResolverService

type Option func(*Service)

// WithResolverFactory injects a custom resolver constructor (for testing).
func WithResolverFactory(f ResolverFactory) Option {
	return func(s *Service) { s.newResolver = f }
}

type Service struct {
	base        *config.TldxConfigOptions
	newResolver ResolverFactory
	presetNames []string
	presetsHelp string
}

// NewService reads the user config file the same way cmd/root.go does, so a
// preset or default that works on the command line works here too.
func NewService(opts ...Option) *Service {
	base := config.NewTldxContext().Config
	base.MaxDomainLength = defaultMaxDomainLength

	if cfg, err := userconfig.Load(); err != nil {
		slog.Warn("Could not load user config", "error", err)
	} else {
		for name, entry := range cfg.Presets {
			presets.TLDs.Override(name, entry.TLDs)
		}
		// nil isSet: there are no command-line flags here, tool arguments are
		// layered on per call instead.
		cfg.Defaults.ApplyTo(base, nil)
	}
	if base.OnlyForSale {
		base.CheckForSale = true
	}

	s := &Service{
		base:        base,
		newResolver: resolver.NewResolverService,
	}
	for _, opt := range opts {
		opt(s)
	}

	s.presetNames, s.presetsHelp = describePresets()
	return s
}

func New(version string, opts ...Option) *server.MCPServer {
	svc := NewService(opts...)

	s := server.NewMCPServer("tldx", version,
		server.WithToolCapabilities(false),
		server.WithInstructions(instructions),
	)

	s.AddTool(svc.checkDomainsTool(), svc.handleCheckDomains)
	s.AddTool(svc.generateAndCheckTool(), svc.handleGenerateAndCheck)

	return s
}

const instructions = `tldx checks domain-name availability over RDAP, with WHOIS and DNS fallbacks.

Two tools:
  check_domains       you already know the exact names to test
  generate_and_check  build names from keywords x prefixes x suffixes x TLDs, then test them

Both are read-only and make no changes to anything.

One call resolves at most ` + maxDomainsLiteral + ` domains. To search a space larger than that,
set only_available=true together with limit=N: the sweep stops as soon as N
available domains are found, so a wide search costs little. Use dry_run=true on
generate_and_check to see exactly which domains a set of arguments would produce
before spending any network requests.

A result whose status is "unknown" means the lookup failed. It does not mean the
domain is free. Only status "available" means available.`

const maxDomainsLiteral = "1000"

// context returns a config owned by one call. The slices are cloned because
// the composer rewrites Config.TLDs in place.
func (s *Service) context() *config.TldxContext {
	cfg := *s.base
	cfg.TLDs = slices.Clone(s.base.TLDs)
	cfg.Prefixes = slices.Clone(s.base.Prefixes)
	cfg.Suffixes = slices.Clone(s.base.Suffixes)
	return &config.TldxContext{Config: &cfg}
}

// describePresets builds the enum values and summary shown in the tld_preset
// description. Runs at startup, after user overrides are applied.
func describePresets() (names []string, help string) {
	all := presets.TLDs.All()

	names = make([]string, 0, len(all)+1)
	for name := range all {
		names = append(names, name)
	}
	sort.Strings(names)

	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, fmt.Sprintf("%s (%d)", name, len(all[name])))
	}

	// "all" is not stored; the composer expands it to every known TLD.
	names = append(names, "all")

	return names, strings.Join(parts, ", ")
}

func readOnlyAnnotations(title string) []mcp.ToolOption {
	return []mcp.ToolOption{
		mcp.WithTitleAnnotation(title),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithOutputSchema[CheckResponse](),
	}
}

func (s *Service) checkDomainsTool() mcp.Tool {
	opts := []mcp.ToolOption{
		mcp.WithDescription(`Check whether specific domain names are registered.

Pass the exact names you want tested, one or many; they are checked in parallel.
Use this when you already know the names. To invent names from keywords, use
generate_and_check instead.

Each result carries a status: "available", "taken", or "unknown" (the lookup
failed, which is NOT the same as available).`),
		mcp.WithArray("domains",
			mcp.Required(),
			mcp.Description(`Fully-qualified domain names, e.g. ["stripe.com","stripe.io"]. Checked exactly as given, with no permutation.`),
			mcp.WithStringItems(mcp.MinLength(1), mcp.MaxLength(253)),
			mcp.MinItems(1),
			mcp.MaxItems(500),
		),
	}
	opts = append(opts, forSaleParams()...)
	opts = append(opts, readOnlyAnnotations("Check specific domains")...)

	return mcp.NewTool("check_domains", opts...)
}

func (s *Service) generateAndCheckTool() mcp.Tool {
	opts := []mcp.ToolOption{
		mcp.WithDescription(`Generate domain-name candidates from keywords and check which are available.

Every keyword is combined with every prefix, suffix, and TLD:
  keyword="stripe", prefixes=["get"], suffixes=["ly"], tlds=["com","io"]
  -> stripe.com stripe.io getstripe.com getstripe.io stripely.com stripely.io
     getstripely.com getstripely.io

HOW MANY DOMAINS A CALL COSTS
  domains = keywords x (1 + prefixes + suffixes + prefixes*suffixes) x tlds

One call resolves at most ` + maxDomainsLiteral + ` domains and is rejected above that, before any
lookup happens. Two ways to stay inside the budget:

  1. Set only_available=true and limit=N. The sweep stops as soon as N available
     domains are found, so a broad search is cheap. This is the best option when
     you just want good names rather than a complete report.
  2. Set dry_run=true first. It returns the exact count and domain list for free,
     with no network requests, so you can size a call before making it.

All parameters are named. Do NOT pass them as positional array items.`),
		mcp.WithArray("keywords",
			mcp.Required(),
			mcp.Description(`Base words to build names from, e.g. ["stripe","atlas"]. Give the bare word, not a domain: a keyword containing a dot has its TLD stripped and added to the TLD list for every other keyword too.`),
			mcp.WithStringItems(mcp.MinLength(1), mcp.MaxLength(63)),
			mcp.MinItems(1),
			mcp.MaxItems(25),
		),
		mcp.WithArray("tlds",
			mcp.Description(`TLDs to check, e.g. ["com","io","ai"]. Defaults to ["com"] when neither tlds nor tld_preset is given.`),
			mcp.WithStringItems(),
			mcp.MaxItems(50),
		),
		mcp.WithArray("prefixes",
			mcp.Description(`Words prepended to each keyword, e.g. ["get","use","my"]. Multiplies the domain count.`),
			mcp.WithStringItems(),
			mcp.MaxItems(10),
		),
		mcp.WithArray("suffixes",
			mcp.Description(`Words appended to each keyword, e.g. ["ly","hub","ify"]. Multiplies the domain count.`),
			mcp.WithStringItems(),
			mcp.MaxItems(10),
		),
		mcp.WithString("tld_preset",
			mcp.Description(`A named set of TLDs, used in addition to any explicit tlds. The number in parentheses is how many TLDs each expands to; check it against the budget. "all" expands to every known TLD and will exceed the budget unless limit is set. Available: `+s.presetsHelp),
			mcp.Enum(s.presetNames...),
		),
		mcp.WithBoolean("dry_run",
			mcp.Description(`When true, return the exact domain list and count these arguments would produce WITHOUT making any network request. Free. Use it to size a call, or to see which TLDs a tld_preset expands to.`),
		),
		mcp.WithBoolean("only_available",
			mcp.Description(`When true, omit taken domains from the results. Pair with limit for a cheap wide search. Default false.`),
		),
		mcp.WithNumber("limit",
			mcp.Description(`Stop the whole sweep once this many available domains are found. 0 means no limit. Setting this is how you afford a large keyword/TLD space.`),
		),
		mcp.WithNumber("max_domain_length",
			mcp.Description(`Skip candidates longer than this many characters, including the TLD. Applied before checking, so it lowers the domain count. Default 64.`),
		),
	}
	opts = append(opts, forSaleParams()...)
	opts = append(opts, readOnlyAnnotations("Generate and check domain names")...)

	return mcp.NewTool("generate_and_check", opts...)
}

func forSaleParams() []mcp.ToolOption {
	return []mcp.ToolOption{
		mcp.WithBoolean("check_for_sale",
			mcp.Description(`When true, look up the RFC 10023 "_for-sale" TXT record of any domain that is taken. Those results gain a "for_sale" object with the asking price, contact URI, and any free text the holder published. Costs one extra DNS query per taken domain. Default false.`),
		),
		mcp.WithBoolean("only_for_sale",
			mcp.Description(`When true, return only taken domains that advertise themselves for sale. Implies check_for_sale. Default false.`),
		),
	}
}
