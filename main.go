package main

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/brandonyoungdev/tldx/cmd"
	"github.com/brandonyoungdev/tldx/internal/config"
)

func main() {
	app := config.NewTldxContext()

	rootCmd := cmd.NewRootCmd(app)

	// Use Fang for graceful shutdown handling
	if err := fang.Execute(
		context.Background(),
		rootCmd,
		fang.WithNotifySignal(os.Interrupt),
	); err != nil {
		os.Exit(1)
	}
}
