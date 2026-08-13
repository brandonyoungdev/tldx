package cmd

import (
	"context"
	"os"

	"github.com/brandonyoungdev/tldx/internal/mcpserver"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

func NewMCPCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Start an MCP (Model Context Protocol) server over stdio",
		Long: `Start a Model Context Protocol (MCP) server that exposes tldx
capabilities as structured tools for AI agents and IDE extensions.

The server communicates over stdin/stdout and exposes two tools:

  check_domains       Check specific domain names you already know
  generate_and_check  Build candidates from keywords, prefixes, suffixes and
                      TLDs, then check them

Both are read-only, return the same result shape, and accept check_for_sale /
only_for_sale to read the RFC 10023 "_for-sale" record of taken domains.

One call resolves at most 1000 domains. generate_and_check takes dry_run to
preview a call's cost with no network requests, and limit to stop a wide sweep
once enough available domains are found.

Your custom TLD presets and [defaults] from the tldx config file apply here too.

Configure your MCP client to run: tldx mcp`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPServer(cmd.Context(), version)
		},
	}
}

func runMCPServer(ctx context.Context, version string) error {
	stdioServer := server.NewStdioServer(mcpserver.New(version))
	return stdioServer.Listen(ctx, os.Stdin, os.Stdout)
}
