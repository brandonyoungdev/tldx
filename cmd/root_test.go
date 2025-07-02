package cmd_test

import (
	"testing"

	"github.com/brandonyoungdev/tldx/cmd"
	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestRootCommandRuns(t *testing.T) {
	app := config.NewTldxContext()

	cmd := cmd.NewRootCmd(app)
	cmd.SetArgs([]string{"google"})

	err := cmd.Execute()
	assert.NoError(t, err)
}
