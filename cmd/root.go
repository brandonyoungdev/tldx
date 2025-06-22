package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/brandonyoungdev/tldx/internal/domain"
	"github.com/spf13/cobra"
)

var Version = "dev"

func init() {
	rootCmd.Flags().StringSliceVarP(&domain.Config.TLDs, "tlds", "t", []string{}, "TLDs to check (e.g. com,io,ai)")
	rootCmd.Flags().StringSliceVarP(&domain.Config.Prefixes, "prefixes", "p", []string{}, "Prefixes to add (e.g. get,my,use)")
	rootCmd.Flags().StringSliceVarP(&domain.Config.Suffixes, "suffixes", "s", []string{}, "Suffixes to add (e.g. ify,ly)")
	rootCmd.Flags().BoolVarP(&domain.Config.Verbose, "verbose", "v", false, "Show verbose output")
	rootCmd.Flags().BoolVarP(&domain.Config.OnlyAvailable, "only-available", "a", false, "Show only available domains")
	rootCmd.Flags().IntVarP(&domain.Config.MaxDomainLength, "max-domain-length", "m", 64, "Maximum length of domain name")
	rootCmd.Flags().BoolVar(&domain.Config.ShowStats, "show-stats", false, "Show statistics at the end of execution")
	rootCmd.Flags().StringVar(&domain.Config.TLDPreset, "tld-preset", "", "Use a tld preset (e.g. popular, tech)")
	rootCmd.Flags().StringVarP(&domain.Config.OutputFormat, "format", "f", "text", "Format of output (text, json, json-stream, json-array, csv)")
	rootCmd.Flags().BoolVar(&domain.Config.NoColor, "no-color", false, "Disable colored output")
	rootCmd.AddCommand(showPresetsCmd)
}

var rootCmd = &cobra.Command{
	Use:     "tldx [keywords]",
	Short:   "Domain availability checker and ideation tool",
	Version: Version,
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if domain.Config.MaxDomainLength <= 0 {
			slog.Error("Invalid max-domain-length provided. Pick a positive number please.")
			return
		}
		if domain.Config.OutputFormat == "" {
			if domain.Config.Verbose {
				// This is okay, since it'll output text by default.
				fmt.Println("Unknown output format. Defaulting to text.")
			}
			domain.Config.OutputFormat = "text"
		}

		domain.Exec(args)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
