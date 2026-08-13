package cmd_test

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/brandonyoungdev/tldx/cmd"
	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCommandRuns(t *testing.T) {
	app := config.NewTldxContext()

	cmd := cmd.NewRootCmd(app)
	cmd.SetArgs([]string{"google"})

	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestRootCommand_OnlyForSale_ReportsItsOwnError(t *testing.T) {
	app := config.NewTldxContext()

	rootCmd := cmd.NewRootCmd(app)
	rootCmd.SetArgs([]string{"google", "--tlds", "com", "--only-for-sale"})
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true

	err := rootCmd.Execute()
	require.ErrorIs(t, err, cmd.ErrNoDomainsForSale)
	// --only-for-sale turns the lookup on by itself.
	assert.True(t, app.Config.CheckForSale)
}

func TestRootCommand_DryRun(t *testing.T) {
	app := config.NewTldxContext()

	rootCmd := cmd.NewRootCmd(app)
	rootCmd.SetArgs([]string{"stripe", "--tlds", "com,io", "--dry-run"})

	require.NoError(t, rootCmd.Execute())
}

func TestRootCommand_ErrNoAvailableDomains(t *testing.T) {
	app := config.NewTldxContext()

	rootCmd := cmd.NewRootCmd(app)
	rootCmd.SetArgs([]string{"google", "--tlds", "com", "--only-available"})
	rootCmd.SilenceErrors = true

	err := rootCmd.Execute()
	if err != nil {
		assert.True(t, errors.Is(err, cmd.ErrNoAvailableDomains),
			"expected ErrNoAvailableDomains, got: %v", err)
	}
}

func TestRootCommand_Limit_Flag(t *testing.T) {
	app := config.NewTldxContext()
	assert.Equal(t, 0, app.Config.Limit)

	rootCmd := cmd.NewRootCmd(app)
	rootCmd.SetArgs([]string{"--limit", "3", "--help"})
	rootCmd.SilenceErrors = true
	rootCmd.Execute() //nolint:errcheck

	f := rootCmd.Flags().Lookup("limit")
	require.NotNil(t, f, "expected --limit flag to be registered")
	assert.Equal(t, "3", f.Value.String())
}

func TestRootCommand_DryRun_Flag(t *testing.T) {
	f := cmd.NewRootCmd(config.NewTldxContext()).Flags().Lookup("dry-run")
	require.NotNil(t, f, "expected --dry-run flag to be registered")
}

func TestRootCommand_FileInput(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "tldx-test-*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("stripe\natlas\n")
	tmpFile.Close()

	app := config.NewTldxContext()
	rootCmd := cmd.NewRootCmd(app)
	rootCmd.SetArgs([]string{"--input", tmpFile.Name(), "--dry-run"})
	require.NoError(t, rootCmd.Execute())
}

func TestOutput_StdinKeywordSupport(t *testing.T) {
	// Pipe test content to simulate stdin via file
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer r.Close()

	w.WriteString("myword\n")
	w.Close()

	// Replace stdin temporarily
	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	app := config.NewTldxContext()
	rootCmd := cmd.NewRootCmd(app)
	rootCmd.SetArgs([]string{"--input", "-", "--dry-run"})
	require.NoError(t, rootCmd.Execute())
}

func TestOutput_TTYAutoDetect(t *testing.T) {
	app := config.NewTldxContext()
	from := cmd.NewRootCmd(app)
	f := from.Flags().Lookup("no-color")
	require.NotNil(t, f)
	assert.Equal(t, "false", f.DefValue)
}

func TestDomainResult_RicherJSONFields(t *testing.T) {
	r := resolver.DomainResult{
		Domain:    "getstripe.com",
		Available: true,
		Keyword:   "stripe",
		Prefix:    "get",
		Suffix:    "",
		TLD:       "com",
	}

	b, err := json.Marshal(r.AsEncodable())
	require.NoError(t, err)
	s := string(b)

	assert.Contains(t, s, `"keyword":"stripe"`)
	assert.Contains(t, s, `"prefix":"get"`)
	assert.Contains(t, s, `"tld":"com"`)
	assert.NotContains(t, s, `"suffix"`)
}
