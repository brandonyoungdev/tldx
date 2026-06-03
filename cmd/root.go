package cmd

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/brandonyoungdev/tldx/internal/config"
	"github.com/brandonyoungdev/tldx/internal/domain"
	"github.com/brandonyoungdev/tldx/internal/input"
	"github.com/brandonyoungdev/tldx/internal/presets"
	"github.com/brandonyoungdev/tldx/internal/userconfig"
	"github.com/spf13/cobra"
)

var Version = "dev"

var ErrNoAvailableDomains = errors.New("no available domains found")

func NewRootCmd(app *config.TldxContext) *cobra.Command {
	asciiArt := `
  _   _     _      
 | | | |   | |     
 | |_| | __| |_  __
 | __| |/ _  \ \/ /
 | |_| | (_| |>  < 
  \__|_|\__,_/_/\_\
`
	cmd := &cobra.Command{
		Use:          "tldx [keywords]",
		Short:        "Domain availability checker and ideation tool",
		Long:         asciiArt + "\nDomain availability checker and ideation tool",
		Version:      Version,
		Args:         cobra.MinimumNArgs(0),
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := userconfig.Load()
			if err != nil {
				slog.Warn("Could not load user presets", "error", err)
				return nil
			}
			for name, entry := range cfg.Presets {
				presets.TLDs.Override(name, entry.TLDs)
			}
			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if app.Config.MaxDomainLength <= 0 {
				slog.Error("Invalid max-domain-length provided. Pick a positive number please.")
				return fmt.Errorf("invalid max-domain-length: must be a positive number")
			}
			if app.Config.OutputFormat == "" {
				if app.Config.Verbose {
					fmt.Println("Unknown output format. Defaulting to text.")
				}
				app.Config.OutputFormat = "text"
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if app.Config.InputFile != "" {
				keywords, err := input.ReadKeywordsFromFile(app.Config.InputFile)
				if err != nil {
					slog.Error("Failed to read keywords from input file", "error", err)
					return err
				}
				args = append(args, keywords...)
			}

			if len(args) == 0 {
				cmd.Help()
				return nil
			}

			found := domain.Exec(cmd.Context(), app, args)

			if app.Config.OnlyAvailable && !found && !app.Config.DryRun {
				return ErrNoAvailableDomains
			}
			return nil
		},
	}

	bindFlags(cmd, app)
	cmd.AddCommand(NewMCPCmd(Version))
	cmd.AddCommand(NewPresetCmd())
	return cmd
}

func bindFlags(cmd *cobra.Command, app *config.TldxContext) {
	cfg := app.Config
	cmd.Flags().StringSliceVarP(&cfg.TLDs, "tlds", "t", []string{}, "TLDs to check (e.g. com,io,ai)")
	cmd.Flags().StringSliceVarP(&cfg.Prefixes, "prefixes", "p", []string{}, "Prefixes to add (e.g. get,my,use)")
	cmd.Flags().StringSliceVarP(&cfg.Suffixes, "suffixes", "s", []string{}, "Suffixes to add (e.g. ify,ly)")
	cmd.Flags().StringVarP(&cfg.InputFile, "input", "i", "", `File to read keywords from. Use "-" to read from stdin.`)
	cmd.Flags().BoolVarP(&cfg.Verbose, "verbose", "v", false, "Show verbose output")
	cmd.Flags().BoolVarP(&cfg.OnlyAvailable, "only-available", "a", false, "Show only available domains")
	cmd.Flags().IntVarP(&cfg.MaxDomainLength, "max-domain-length", "m", 64, "Maximum length of domain name")
	cmd.Flags().BoolVar(&cfg.ShowStats, "show-stats", false, "Show statistics at the end of execution")
	cmd.Flags().StringVar(&cfg.TLDPreset, "tld-preset", "", "Use a tld preset (e.g. popular, tech)")
	cmd.Flags().StringVarP(&cfg.OutputFormat, "format", "f", "text", "Format of output (text, json, json-stream, json-array, csv, grouped, grouped-tld)")
	cmd.Flags().BoolVar(&cfg.NoColor, "no-color", false, "Disable colored output")
	cmd.Flags().BoolVarP(&cfg.Regex, "regex", "r", false, "Enable regex pattern matching for domain keywords")
	cmd.Flags().IntVarP(&cfg.Limit, "limit", "l", 0, "Stop after finding this many available domains (0 = no limit)")
	cmd.Flags().BoolVar(&cfg.DryRun, "dry-run", false, "Print domains that would be checked without making network calls")
}
