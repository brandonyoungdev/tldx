package main

import (
	"context"
	"errors"
	"os"

	"github.com/brandonyoungdev/tldx/cmd"
	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/charmbracelet/fang"
)

func main() {
	app := config.NewTldxContext()

	rootCmd := cmd.NewRootCmd(app)

	// Use Fang for graceful shutdown handling
	if err := fang.Execute(
		context.Background(),
		rootCmd,
		fang.WithNotifySignal(os.Interrupt),
		fang.WithVersion(cmd.Version),
	); err != nil {
		if errors.Is(err, cmd.ErrNoAvailableDomains) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}
