package cmd_test

import (
	"strings"
	"testing"

	"github.com/brandonyoungdev/tldx/cmd"
	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The server itself is tested in internal/mcpserver; this covers the cobra wiring.

func TestNewMCPCmd_Structure(t *testing.T) {
	mcpCmd := cmd.NewMCPCmd("v1.0.0")
	require.NotNil(t, mcpCmd)
	assert.Equal(t, "mcp", mcpCmd.Use)
	assert.NotEmpty(t, mcpCmd.Short)
	assert.NotEmpty(t, mcpCmd.Long)
}

func TestNewMCPCmd_LongDescribesBothTools(t *testing.T) {
	long := cmd.NewMCPCmd("v1.0.0").Long

	assert.Contains(t, long, "check_domains")
	assert.Contains(t, long, "generate_and_check")
	assert.NotContains(t, long, "list_tld_presets")
	assert.False(t, strings.Contains(long, "check_domain "),
		"the singular check_domain tool no longer exists")
}

func TestRootCmd_RegistersMCPCommand(t *testing.T) {
	root := cmd.NewRootCmd(config.NewTldxContext())

	var found bool
	for _, c := range root.Commands() {
		if c.Name() == "mcp" {
			found = true
		}
	}
	assert.True(t, found, "expected the mcp subcommand to be registered")
}
