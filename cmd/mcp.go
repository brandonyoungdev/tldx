package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/brandonyoungdev/tldx/internal/composer"
	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/presets"
	"github.com/brandonyoungdev/tldx/internal/resolver"
	"github.com/brandonyoungdev/tldx/internal/validate"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

func NewMCPCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Start an MCP (Model Context Protocol) server over stdio",
		Long: `Start a Model Context Protocol (MCP) server that exposes tldx
capabilities as structured tools for AI agents and IDE extensions.

The server communicates over stdin/stdout and exposes four tools:

  check_domain          - Check if a single domain is available
  check_domains         - Check a list of domains in parallel
  generate_and_check    - Build permutations from keywords and check availability
  list_tld_presets      - List all built-in TLD presets

Configure your MCP client to run: tldx mcp`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPServer(cmd.Context(), version)
		},
	}
}

func runMCPServer(ctx context.Context, version string) error {
	s := server.NewMCPServer("tldx", version,
		server.WithToolCapabilities(false),
	)

	s.AddTool(checkDomainTool(), checkDomainHandler)
	s.AddTool(checkDomainsTool(), checkDomainsHandler)
	s.AddTool(generateAndCheckTool(), generateAndCheckHandler)
	s.AddTool(listTLDPresetsTool(), listTLDPresetsHandler)

	stdioServer := server.NewStdioServer(s)
	return stdioServer.Listen(ctx, os.Stdin, os.Stdout)
}

func checkDomainTool() mcp.Tool {
	return mcp.NewTool("check_domain",
		mcp.WithDescription("Check whether a single domain name is available for registration."),
		mcp.WithString("domain",
			mcp.Required(),
			mcp.Description("The fully-qualified domain name to check (e.g. stripe.com)"),
		),
	)
}

func checkDomainsTool() mcp.Tool {
	return mcp.NewTool("check_domains",
		mcp.WithDescription("Check availability for a list of domain names in parallel."),
		mcp.WithArray("domains",
			mcp.Required(),
			mcp.Description("List of fully-qualified domain names to check (e.g. [\"stripe.com\",\"stripe.io\"])"),
			mcp.WithStringItems(),
			mcp.MaxItems(2000),
		),
	)
}

func generateAndCheckTool() mcp.Tool {
	return mcp.NewTool("generate_and_check",
		mcp.WithDescription(`Generate domain name permutations from keywords and check their availability.

Permutations are built by combining each keyword with every prefix, suffix, and TLD.
Example: keyword="stripe", prefixes=["get"], suffixes=["ly"], tlds=["com","io"]
produces: stripe.com, stripe.io, getstripe.com, getstripe.io, stripely.com, stripely.io, getstripely.com, getstripely.io

Each result includes keyword/prefix/suffix/tld metadata so agents can reason about which patterns work best.
All parameters are named — do NOT pass them as positional array items.`),
		mcp.WithArray("keywords",
			mcp.Required(),
			mcp.Description("Base keywords to build domain names from (e.g. [\"stripe\", \"atlas\"]). For very large lists, consider multiple calls to avoid network timeouts."),
			mcp.WithStringItems(mcp.MinLength(1), mcp.MaxLength(63)),
			mcp.MaxItems(500),
		),
		mcp.WithArray("tlds",
			mcp.Description("TLDs to check (e.g. [\"com\",\"io\",\"ai\"]). Defaults to [\"com\"] if omitted."),
			mcp.WithStringItems(),
			mcp.MaxItems(100),
		),
		mcp.WithArray("prefixes",
			mcp.Description("Optional prefixes to prepend to each keyword (e.g. [\"get\",\"use\",\"my\"])."),
			mcp.WithStringItems(),
			mcp.MaxItems(50),
		),
		mcp.WithArray("suffixes",
			mcp.Description("Optional suffixes to append to each keyword (e.g. [\"ly\",\"hub\",\"ify\"])."),
			mcp.WithStringItems(),
			mcp.MaxItems(50),
		),
		mcp.WithString("tld_preset",
			mcp.Description("Use a named TLD preset instead of explicit tlds (e.g. \"popular\", \"tech\", \"startup\"). Run list_tld_presets to see all options."),
		),
		mcp.WithBoolean("only_available",
			mcp.Description("When true, only return domains that are available. Default false."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Stop after finding this many available domains. 0 means no limit. Default 0."),
		),
		mcp.WithNumber("max_domain_length",
			mcp.Description("Skip domains longer than this many characters. Default 64."),
		),
	)
}

func listTLDPresetsTool() mcp.Tool {
	return mcp.NewTool("list_tld_presets",
		mcp.WithDescription("List all available TLD presets. Use preset names with generate_and_check."),
	)
}

var MCPCheckDomainHandler = checkDomainHandler
var MCPCheckDomainsHandler = checkDomainsHandler
var MCPGenerateAndCheckHandler = generateAndCheckHandler
var MCPListTLDPresetsHandler = listTLDPresetsHandler

var MCPCheckDomainTool      = checkDomainTool
var MCPCheckDomainsTool     = checkDomainsTool
var MCPGenerateAndCheckTool = generateAndCheckTool
var MCPListTLDPresetsTool   = listTLDPresetsTool

// ResolverFactory creates a resolver service. Overridable in tests for dependency injection.
var ResolverFactory = func(app *config.TldxContext, opts ...resolver.ResolverOption) *resolver.ResolverService {
	return resolver.NewResolverService(app, opts...)
}

func checkDomainHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain, err := req.RequireString("domain")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if !validate.IsValidDomainOrKeyword(domain) {
		return mcp.NewToolResultError(fmt.Sprintf("invalid domain name: %q", domain)), nil
	}

	app := config.NewTldxContext()
	svc := ResolverFactory(app)
	result, checkErr := svc.CheckDomain(ctx, domain)

	out := map[string]any{
		"domain":    domain,
		"available": !result.Registered,
		"details":   result.Details,
	}
	if checkErr != nil {
		out["error"] = checkErr.Error()
	}

	return toolResultJSON(out)
}

func checkDomainsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domains, err := req.RequireStringSlice("domains")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if len(domains) == 0 {
		return mcp.NewToolResultError("domains list must not be empty"), nil
	}

	app := config.NewTldxContext()
	svc := ResolverFactory(app)

	specs := make([]resolver.DomainSpec, len(domains))
	for i, d := range domains {
		specs[i] = resolver.DomainSpec{Domain: d}
	}

	resultChan := svc.CheckDomainsStreaming(ctx, specs)
	var results []map[string]any
	for r := range resultChan {
		entry := map[string]any{
			"domain":    r.Domain,
			"available": r.Available,
			"details":   r.Details,
		}
		if r.Error != nil {
			entry["error"] = r.Error.Error()
		}
		results = append(results, entry)
	}

	return toolResultJSON(results)
}

func generateAndCheckHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	keywords, err := req.RequireStringSlice("keywords")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if len(keywords) == 0 {
		return mcp.NewToolResultError("keywords list must not be empty"), nil
	}

	app := config.NewTldxContext()
	app.Config.TLDs = req.GetStringSlice("tlds", []string{})
	app.Config.Prefixes = req.GetStringSlice("prefixes", []string{})
	app.Config.Suffixes = req.GetStringSlice("suffixes", []string{})
	app.Config.TLDPreset = req.GetString("tld_preset", "")
	app.Config.OnlyAvailable = req.GetBool("only_available", false)
	limit := req.GetInt("limit", 0)
	maxLen := req.GetInt("max_domain_length", 64)
	if maxLen > 0 {
		app.Config.MaxDomainLength = maxLen
	}

	composerSvc := composer.NewComposerService(app)
	specs, warnings := composerSvc.Compile(keywords)

	if len(warnings) > 0 && len(specs) == 0 {
		msgs := make([]string, len(warnings))
		for i, w := range warnings {
			msgs[i] = w.Error()
		}
		return mcp.NewToolResultError(fmt.Sprintf("compilation warnings: %v", msgs)), nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	resolverSvc := ResolverFactory(app)
	resultChan := resolverSvc.CheckDomainsStreaming(ctx, specs)

	availableCount := 0
	var results []map[string]any

	for r := range resultChan {
		if app.Config.OnlyAvailable && !r.Available {
			continue
		}
		entry := map[string]any{
			"domain":    r.Domain,
			"available": r.Available,
			"keyword":   r.Keyword,
			"prefix":    r.Prefix,
			"suffix":    r.Suffix,
			"tld":       r.TLD,
		}
		if r.Details != "" {
			entry["details"] = r.Details
		}
		if r.Error != nil {
			entry["error"] = r.Error.Error()
		}
		results = append(results, entry)

		if r.Available {
			availableCount++
			if limit > 0 && availableCount >= limit {
				cancel()
				break
			}
		}
	}

	return toolResultJSON(map[string]any{
		"results": results,
		"total":   len(results),
	})
}

func listTLDPresetsHandler(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	all := presets.TLDs.All()
	out := make([]map[string]any, 0, len(all)+1)

	out = append(out, map[string]any{
		"name":        "all",
		"description": "Use all available TLDs",
	})

	for name, tlds := range all {
		out = append(out, map[string]any{
			"name": name,
			"tlds": tlds,
		})
	}

	return toolResultJSON(out)
}

func toolResultJSON(v any) (*mcp.CallToolResult, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to encode result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}
