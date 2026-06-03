package cmd

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/brandonyoungdev/tldx/internal/presets"
	"github.com/brandonyoungdev/tldx/internal/userconfig"
	"github.com/spf13/cobra"
	"golang.org/x/net/publicsuffix"
)

func NewPresetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "preset",
		Short: "Manage custom TLD presets",
		Long:  "Add, remove, and list custom TLD presets that persist between runs.",
	}

	cmd.AddCommand(newPresetAddCmd())
	cmd.AddCommand(newPresetRemoveCmd())
	cmd.AddCommand(newPresetListCmd())
	return cmd
}

func newPresetAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <name> <tld> [tld...]",
		Short: "Add or replace a custom TLD preset",
		Example: `  tldx preset add myteam com io ai
  tldx preset add saas com,io,app,dev`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.ToLower(strings.TrimSpace(args[0]))
			if name == "" {
				return fmt.Errorf("preset name cannot be empty")
			}

			// Remaining args are TLDs; each arg may itself be comma-separated.
			var tlds []string
			var invalid []string
			for _, arg := range args[1:] {
				for _, part := range strings.FieldsFunc(arg, func(r rune) bool { return r == ',' }) {
					p := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(part), "."))
					if p == "" {
						continue
					}
					suffix, icann := publicsuffix.PublicSuffix(p)
					if !icann || suffix != p {
						invalid = append(invalid, p)
					} else {
						tlds = append(tlds, p)
					}
				}
			}
			if len(invalid) > 0 {
				return fmt.Errorf("invalid TLD(s): %s", strings.Join(invalid, ", "))
			}
			if len(tlds) == 0 {
				return fmt.Errorf("at least one TLD must be provided")
			}

			cfg, err := userconfig.Load()
			if err != nil {
				slog.Error("Failed to load user config", "error", err)
				return err
			}

			cfg.Presets[name] = userconfig.PresetEntry{TLDs: tlds}

			if err := userconfig.Save(cfg); err != nil {
				slog.Error("Failed to save user config", "error", err)
				return err
			}

			path, _ := userconfig.ConfigPath()
			cmd.Printf("Saved preset %q (%s) → %s\n", name, strings.Join(tlds, ", "), path)
			return nil
		},
	}
}

func newPresetRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove a custom TLD preset",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.ToLower(strings.TrimSpace(args[0]))

			// Refuse to remove built-in presets
			if _, isBuiltin := presets.DefaultTLDPresets[name]; isBuiltin {
				return fmt.Errorf("%q is a built-in preset and cannot be removed", name)
			}

			cfg, err := userconfig.Load()
			if err != nil {
				slog.Error("Failed to load user config", "error", err)
				return err
			}

			if _, ok := cfg.Presets[name]; !ok {
				return fmt.Errorf("preset %q not found in user config", name)
			}

			delete(cfg.Presets, name)

			if err := userconfig.Save(cfg); err != nil {
				slog.Error("Failed to save user config", "error", err)
				return err
			}

			cmd.Printf("Removed preset %q\n", name)
			return nil
		},
	}
}

func newPresetListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all TLD presets (built-in and custom)",
		Run: func(cmd *cobra.Command, args []string) {
			userCfg, err := userconfig.Load()
			if err != nil {
				slog.Warn("Could not load user config; showing built-in presets only", "error", err)
				userCfg = &userconfig.UserConfig{Presets: map[string]userconfig.PresetEntry{}}
			}

			// Build the combined store so ShowAllPresets renders everything.
			store := presets.NewTypedStore("tld", presets.DefaultTLDPresets)
			for name, entry := range userCfg.Presets {
				store.Override(name, entry.TLDs)
			}

			// Collect user-defined names for annotation.
			userNames := make(map[string]bool, len(userCfg.Presets))
			for name := range userCfg.Presets {
				userNames[name] = true
			}

			all := store.All()
			names := make([]string, 0, len(all))
			for name := range all {
				names = append(names, name)
			}
			sort.Strings(names)

			const maxWidth = 70
			const labelWidth = 24

			cmd.Printf("\nTLD Presets  (* = custom):\n\n")
			cmd.Printf("%-*s  %s\n\n", labelWidth, "all", "(use all available TLDs)")

			for _, name := range names {
				tlds := all[name]
				sortedTLDs := make([]string, len(tlds))
				copy(sortedTLDs, tlds)
				sort.Slice(sortedTLDs, func(i, j int) bool {
					if len(sortedTLDs[i]) != len(sortedTLDs[j]) {
						return len(sortedTLDs[i]) < len(sortedTLDs[j])
					}
					return sortedTLDs[i] < sortedTLDs[j]
				})
				tldStr := strings.Join(sortedTLDs, " ")

				label := name
				if userNames[name] {
					label = name + " *"
				}

				if len(tldStr) > maxWidth-labelWidth-4 {
					lines := wrapPresetText(tldStr, maxWidth-labelWidth-4)
					cmd.Printf("%-*s  %s\n", labelWidth, label, lines[0])
					for i := 1; i < len(lines); i++ {
						cmd.Printf("%-*s  %s\n", labelWidth, "", lines[i])
					}
				} else {
					cmd.Printf("%-*s  %s\n", labelWidth, label, tldStr)
				}
				cmd.Println()
			}

			cmd.Printf("%-*s  %s\n\n", labelWidth, "all", "(use all available TLDs)")

			path, _ := userconfig.ConfigPath()
			cmd.Printf("Config file: %s\n", path)
		},
	}
}

// wrapPresetText wraps text at word boundaries within maxWidth.
func wrapPresetText(text string, maxWidth int) []string {
	if len(text) <= maxWidth {
		return []string{text}
	}

	var lines []string
	words := strings.Split(text, " ")
	currentLine := ""

	for i, word := range words {
		testLine := currentLine
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		if len(testLine) > maxWidth {
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = word
			} else {
				lines = append(lines, word)
			}
		} else {
			currentLine = testLine
		}

		if i == len(words)-1 && currentLine != "" {
			lines = append(lines, currentLine)
		}
	}

	return lines
}
