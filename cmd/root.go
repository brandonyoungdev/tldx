package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "domitool",
	Short: "A CLI tool for researching available domains",
	Long:  `Domitool is a CLI tool for researching available domains.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		slog.Error(fmt.Sprintf("Error: %s", err))
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolP("gui", "u", false, "Gui instead of raw output stdout")
}
