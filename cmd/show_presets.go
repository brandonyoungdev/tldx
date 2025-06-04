package cmd

import (
	"strings"

	"github.com/brandonyoungdev/tldx/internal/presets"
	"github.com/spf13/cobra"
)

var showPresetsCmd = &cobra.Command{
	Use:   "show-tld-presets",
	Short: "Show available TLD presets",
	Run: func(cmd *cobra.Command, args []string) {

		store := presets.NewTypedStore("tld", presets.DefaultTLDPresets)

		presets.ShowAllPresets(store, func(v []string) string {
			return strings.Join(v, ", ")
		})
	},
}
