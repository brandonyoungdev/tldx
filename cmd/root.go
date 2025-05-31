package cmd

import (
	"log/slog"
	"os"

	"github.com/brandonyoungdev/tldx/internal/domain"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.Flags().StringSliceVarP(&domain.Config.TLDs, "tlds", "t", []string{}, "TLDs to check (e.g. com,io,dev)")
	rootCmd.Flags().StringSliceVarP(&domain.Config.Prefixes, "prefixes", "p", []string{}, "Prefixes to add (e.g. get,my,use)")
	rootCmd.Flags().StringSliceVarP(&domain.Config.Suffixes, "suffixes", "s", []string{}, "Suffixes to add (e.g. ify,ly)")
	rootCmd.Flags().IntVarP(&domain.Config.MaxDomainLength, "max-domain-length", "m", 64, "Maximum length of domain name")
}

var rootCmd = &cobra.Command{
	Use:   "tldx [keywords]",
	Short: "Domain availability checker and ideation tool",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if domain.Config.MaxDomainLength <= 0 {
			slog.Error("Invalid max-domain-length provided")
			return
		}
		domain.Exec(args)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
