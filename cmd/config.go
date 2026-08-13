package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/brandonyoungdev/tldx/internal/userconfig"
	"github.com/spf13/cobra"
)

const configTemplate = `# tldx configuration
#
# Values under [defaults] are applied to every run. Any flag passed on the
# command line overrides the matching value here.

[defaults]
# TLDs checked when neither --tlds nor --tld-preset is given.
# tlds = ["com", "se", "nu"]

# A preset name instead of (or alongside) an explicit tlds list.
# Run "tldx preset list" to see the available names.
# tld_preset = "popular"

# prefixes = ["get", "my"]
# suffixes = ["ly", "hq"]
# max_domain_length = 64
# format = "text"
# limit = 0
# only_available = true

# Check taken domains for an RFC 10023 "_for-sale" record. Costs one extra DNS
# query per taken domain. only_for_sale implies for_sale.
# for_sale = true
# only_for_sale = true

# show_stats = true
# no_color = false
# verbose = false

# Custom presets, usable via --tld-preset <name>.
# Add them here by hand or with "tldx preset add <name> <tld>...".
# [presets.nordic]
# tlds = ["se", "nu", "no", "dk", "fi"]
`

func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and manage the tldx config file",
		Long: "Show, locate, and create the config file holding your default flags and custom presets.\n" +
			"Set TLDX_CONFIG to use a different file.",
	}

	cmd.AddCommand(newConfigPathCmd())
	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigInitCmd())
	return cmd
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the path of the config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := userconfig.ConfigPath()
			if err != nil {
				return err
			}
			cmd.Println(path)
			return nil
		},
	}
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show the configured defaults",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := userconfig.ConfigPath()
			if err != nil {
				return err
			}

			cfg, err := userconfig.Load()
			if err != nil {
				slog.Error("Failed to load user config", "error", err)
				return err
			}

			if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
				cmd.Printf("No config file yet (%s)\n", path)
				cmd.Println("Run \"tldx config init\" to create one.")
				return nil
			}

			cmd.Printf("Config file: %s\n\n", path)

			lines := describeDefaults(cfg.Defaults)
			if len(lines) == 0 {
				cmd.Println("Defaults: none set")
			} else {
				cmd.Println("Defaults:")
				for _, line := range lines {
					cmd.Printf("  %s\n", line)
				}
			}

			cmd.Printf("\nCustom presets: %d (run \"tldx preset list\" to see them)\n", len(cfg.Presets))
			return nil
		},
	}
}

func newConfigInitCmd() *cobra.Command {
	var force bool

	c := &cobra.Command{
		Use:   "init",
		Short: "Create a commented config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := userconfig.ConfigPath()
			if err != nil {
				return err
			}

			if _, err := os.Stat(path); err == nil && !force {
				return fmt.Errorf("config file already exists at %s (use --force to overwrite)", path)
			}

			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return fmt.Errorf("create config dir: %w", err)
			}
			if err := os.WriteFile(path, []byte(configTemplate), 0o644); err != nil {
				return fmt.Errorf("write %s: %w", path, err)
			}

			cmd.Printf("Wrote config template → %s\n", path)
			return nil
		},
	}

	c.Flags().BoolVar(&force, "force", false, "Overwrite an existing config file")
	return c
}

func describeDefaults(d userconfig.Defaults) []string {
	var out []string

	add := func(name, value string) {
		out = append(out, fmt.Sprintf("%-18s %s", name, value))
	}

	if len(d.TLDs) > 0 {
		add("tlds", strings.Join(d.TLDs, ", "))
	}
	if d.TLDPreset != "" {
		add("tld_preset", d.TLDPreset)
	}
	if len(d.Prefixes) > 0 {
		add("prefixes", strings.Join(d.Prefixes, ", "))
	}
	if len(d.Suffixes) > 0 {
		add("suffixes", strings.Join(d.Suffixes, ", "))
	}
	if d.MaxDomainLength != nil {
		add("max_domain_length", fmt.Sprintf("%d", *d.MaxDomainLength))
	}
	if d.Format != "" {
		add("format", d.Format)
	}
	if d.Limit != nil {
		add("limit", fmt.Sprintf("%d", *d.Limit))
	}
	if d.OnlyAvailable {
		add("only_available", "true")
	}
	if d.ForSale {
		add("for_sale", "true")
	}
	if d.OnlyForSale {
		add("only_for_sale", "true")
	}
	if d.ShowStats {
		add("show_stats", "true")
	}
	if d.NoColor {
		add("no_color", "true")
	}
	if d.Verbose {
		add("verbose", "true")
	}

	return out
}
