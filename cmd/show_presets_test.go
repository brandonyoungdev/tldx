package cmd_test

import (
	"testing"

	"github.com/brandonyoungdev/tldx/cmd"
	"github.com/stretchr/testify/assert"
)

func TestShowPresetsCmd(t *testing.T) {
	cmd := cmd.NewShowPresetsCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.NoError(t, err)
}
