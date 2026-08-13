package cmd_test

import (
	"context"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/brandonyoungdev/tldx/cmd"
)

// The stdio server reads os.Stdin, so this only asserts that running the
// command hands off to it and returns once the context is done.
func TestMCPCmd_StopsWithTheContext(t *testing.T) {
	t.Setenv("TLDX_CONFIG", filepath.Join(t.TempDir(), "config.toml"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mcpCmd := cmd.NewMCPCmd("test")
	mcpCmd.SetArgs([]string{})
	mcpCmd.SetOut(io.Discard)
	mcpCmd.SetErr(io.Discard)

	done := make(chan error, 1)
	go func() { done <- mcpCmd.ExecuteContext(ctx) }()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("mcp command did not return after its context was cancelled")
	}
}
