package cmd

import (
	"fmt"
	"log/slog"

	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/domain"
	"github.com/brandonyoungdev/tldx/internal/input"
	"github.com/spf13/cobra"
)

var Version = "dev"

func NewRootCmd(app *config.TldxContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tldx [keywords]",
		Short:   "Domain availability checker and ideation tool",
		Version: Version,
		Args:    cobra.MinimumNArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			if app.Config.MaxDomainLength <= 0 {
				slog.Error("Invalid max-domain-length provided. Pick a positive number please.")
				return
			}
			if app.Config.OutputFormat == "" {
				if app.Config.Verbose {
					fmt.Println("Unknown output format. Defaulting to text.")
				}
				app.Config.OutputFormat = "text"
			}

			if app.Config.InputFile != "" {
				keywords, err := input.ReadKeywordsFromFile(app.Config.InputFile)
				if err != nil {
					slog.Error("Failed to read keywords from input file", "error", err)
					return
				}
				args = append(args, keywords...)
			}

		},
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				slog.Error("No keywords provided. Please provide keywords to check.")
				cmd.Help()
				return
			}

			domain.Exec(cmd.Context(), app, args)
		},
	}

	bindFlags(cmd, app)
	cmd.AddCommand(NewShowPresetsCmd())
	return cmd
}

func bindFlags(cmd *cobra.Command, app *config.TldxContext) {
	cfg := app.Config
	cmd.Flags().StringSliceVarP(&cfg.TLDs, "tlds", "t", []string{}, "TLDs to check (e.g. com,io,ai)")
	cmd.Flags().StringSliceVarP(&cfg.Prefixes, "prefixes", "p", []string{}, "Prefixes to add (e.g. get,my,use)")
	cmd.Flags().StringSliceVarP(&cfg.Suffixes, "suffixes", "s", []string{}, "Suffixes to add (e.g. ify,ly)")
	cmd.Flags().StringVarP(&cfg.InputFile, "input", "i", "", "File to read keywords from.")
	cmd.Flags().BoolVarP(&cfg.Verbose, "verbose", "v", false, "Show verbose output")
	cmd.Flags().BoolVarP(&cfg.OnlyAvailable, "only-available", "a", false, "Show only available domains")
	cmd.Flags().IntVarP(&cfg.MaxDomainLength, "max-domain-length", "m", 64, "Maximum length of domain name")
	cmd.Flags().BoolVar(&cfg.ShowStats, "show-stats", false, "Show statistics at the end of execution")
	cmd.Flags().StringVar(&cfg.TLDPreset, "tld-preset", "", "Use a tld preset (e.g. popular, tech)")
	cmd.Flags().StringVarP(&cfg.OutputFormat, "format", "f", "text", "Format of output (text, json, json-stream, json-array, csv)")
	cmd.Flags().BoolVar(&cfg.NoColor, "no-color", false, "Disable colored output")
}
